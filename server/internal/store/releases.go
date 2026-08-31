package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrReleaseConflict = errors.New("release conflict")
	ErrInvalidRelease  = errors.New("invalid release")
)

// AppRelease 应用发布记录（Ops 可见全字段；公开 API 用 PublicRelease）。
type AppRelease struct {
	ID                  string  `json:"id"`
	Product             string  `json:"product"`
	Version             string  `json:"version"`
	Channel             string  `json:"channel"`
	Platform            string  `json:"platform"`
	Arch                string  `json:"arch"`
	MinSupportedVersion string  `json:"minSupportedVersion,omitempty"`
	ForceUpdate         bool    `json:"forceUpdate"`
	Title               string  `json:"title"`
	NotesMd             string  `json:"notesMd"`
	DownloadURL         string  `json:"downloadUrl"`
	SHA256              string  `json:"sha256"`
	FileSize            int64   `json:"fileSize"`
	Status              string  `json:"status"`
	PublishedAt         *string `json:"publishedAt,omitempty"`
	CreatedBy           string  `json:"createdBy,omitempty"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

// PublicNotesExcerptRunes 桌面更新框与官网默认展示的更新说明上限（按 Unicode 字计）。
const PublicNotesExcerptRunes = 800

// NotesExcerpt 把更新说明裁成公开摘要，避免对话框被整篇 changelog 撑爆。
// 优先在段落或句末断开；truncated 表示还有全文。
func NotesExcerpt(md string) (excerpt string, truncated bool) {
	md = strings.TrimSpace(md)
	if md == "" {
		return "", false
	}
	if utf8.RuneCountInString(md) <= PublicNotesExcerptRunes {
		return md, false
	}
	runes := []rune(md)
	cut := string(runes[:PublicNotesExcerptRunes])
	s := cut
	if i := strings.LastIndex(cut, "\n\n"); i >= PublicNotesExcerptRunes/3 {
		s = strings.TrimSpace(cut[:i])
	} else if i := strings.LastIndexAny(cut, "\n。！？"); i >= PublicNotesExcerptRunes/3 {
		s = strings.TrimSpace(cut[:i+1])
	} else {
		s = strings.TrimSpace(cut)
	}
	if s == "" {
		s = strings.TrimSpace(cut)
	}
	return s + "…", true
}

// PublicRelease 公开面：不含 createdBy 等内部字段。
// notesMd 仍是全文；notesExcerpt / notesTruncated 供桌面与官网默认折叠。
func PublicRelease(r *AppRelease) map[string]any {
	if r == nil {
		return nil
	}
	excerpt, truncated := NotesExcerpt(r.NotesMd)
	out := map[string]any{
		"product":             r.Product,
		"channel":             r.Channel,
		"platform":            r.Platform,
		"arch":                r.Arch,
		"version":             r.Version,
		"title":               r.Title,
		"notesMd":             r.NotesMd,
		"notesExcerpt":        excerpt,
		"notesTruncated":      truncated,
		"downloadUrl":         r.DownloadURL,
		"sha256":              r.SHA256,
		"fileSize":            r.FileSize,
		"minSupportedVersion": r.MinSupportedVersion,
		"forceUpdate":         r.ForceUpdate,
	}
	if r.PublishedAt != nil {
		out["publishedAt"] = *r.PublishedAt
	}
	return out
}

// PublicReleaseCard 公开列表条目：不含全文、下载地址和校验和，避免客户端 limit 被用来批量扒说明。
func PublicReleaseCard(r *AppRelease) map[string]any {
	if r == nil {
		return nil
	}
	out := map[string]any{
		"product":  r.Product,
		"channel":  r.Channel,
		"platform": r.Platform,
		"arch":     r.Arch,
		"version":  r.Version,
		"title":    r.Title,
	}
	if r.PublishedAt != nil {
		out["publishedAt"] = *r.PublishedAt
	}
	return out
}

type ReleaseListFilter struct {
	Product  string
	Channel  string
	Platform string
	Arch     string
	Status   string
	Version  string
	Limit    int
	Offset   int
}

func (s *Store) InsertRelease(ctx context.Context, r *AppRelease) error {
	force := 0
	if r.ForceUpdate {
		force = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_releases (
			id, product_code, version_label, channel_code, platform_code, arch_code,
			min_supported_version, force_update, title_text, notes_md, download_url,
			sha256_hex, file_size, release_status, published_at, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Product, r.Version, r.Channel, r.Platform, r.Arch,
		r.MinSupportedVersion, force, r.Title, r.NotesMd, r.DownloadURL,
		r.SHA256, r.FileSize, r.Status, toDBNullTime(r.PublishedAt), r.CreatedBy,
		toDBTime(r.CreatedAt), toDBTime(r.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrReleaseConflict
		}
		return err
	}
	return nil
}

func (s *Store) UpdateRelease(ctx context.Context, r *AppRelease) error {
	force := 0
	if r.ForceUpdate {
		force = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE app_releases SET
			min_supported_version = ?, force_update = ?, title_text = ?, notes_md = ?,
			download_url = ?, sha256_hex = ?, file_size = ?, release_status = ?,
			published_at = ?, updated_at = ?
		WHERE id = ?`,
		r.MinSupportedVersion, force, r.Title, r.NotesMd,
		r.DownloadURL, r.SHA256, r.FileSize, r.Status,
		toDBNullTime(r.PublishedAt), toDBTime(r.UpdatedAt), r.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReleaseNotesPatch 同一版本多内核共用的更新说明字段。
type ReleaseNotesPatch struct {
	Title   *string
	NotesMd *string
}

// UpdateReleaseNotesByVersion 把同一产品+版本下未撤回记录的标题/更新说明一并改掉。
// 下载地址、校验和等仍按单条内核维护。
func (s *Store) UpdateReleaseNotesByVersion(ctx context.Context, product, version, updatedAt string, patch ReleaseNotesPatch) (int, error) {
	product = strings.TrimSpace(product)
	version = strings.TrimSpace(version)
	if product == "" || version == "" {
		return 0, ErrInvalidRelease
	}
	if patch.Title == nil && patch.NotesMd == nil {
		return 0, nil
	}
	set := []string{"updated_at = ?"}
	args := []any{toDBTime(updatedAt)}
	if patch.Title != nil {
		set = append(set, "title_text = ?")
		args = append(args, *patch.Title)
	}
	if patch.NotesMd != nil {
		set = append(set, "notes_md = ?")
		args = append(args, *patch.NotesMd)
	}
	args = append(args, product, version)
	res, err := s.db.ExecContext(ctx, `
		UPDATE app_releases SET `+strings.Join(set, ", ")+`
		WHERE product_code = ? AND version_label = ? AND release_status <> 'yanked'`,
		args...,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *Store) GetReleaseByID(ctx context.Context, id string) (*AppRelease, error) {
	row := s.db.QueryRowContext(ctx, releaseSelect+` WHERE id = ?`, id)
	return scanRelease(row)
}

// GetReleaseByDim 按产品维度 + 版本查找唯一发布记录。
func (s *Store) GetReleaseByDim(ctx context.Context, product, channel, platform, arch, version string) (*AppRelease, error) {
	row := s.db.QueryRowContext(ctx, releaseSelect+`
		WHERE product_code = ? AND channel_code = ? AND platform_code = ? AND arch_code = ? AND version_label = ?`,
		product, channel, platform, arch, version,
	)
	return scanRelease(row)
}

func (s *Store) ListReleases(ctx context.Context, f ReleaseListFilter) ([]AppRelease, int, error) {
	limit, offset := clampPage(f.Limit, f.Offset)
	where := []string{"1=1"}
	args := []any{}
	appendEq(&where, &args, "product_code", f.Product)
	appendEq(&where, &args, "channel_code", f.Channel)
	appendEq(&where, &args, "platform_code", f.Platform)
	appendEq(&where, &args, "arch_code", f.Arch)
	appendEq(&where, &args, "version_label", f.Version)
	if st := strings.TrimSpace(f.Status); st != "" && st != "all" {
		where = append(where, "release_status = ?")
		args = append(args, st)
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_releases WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := releaseSelect + ` WHERE ` + clause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanReleaseRows(rows)
	return items, total, err
}

// ListPublishedReleases 仅 published，供公开 changelog。limit/offset 走 clampPage（默认 20，上限 50）。
func (s *Store) ListPublishedReleases(ctx context.Context, product, channel, platform, arch string, limit, offset int) ([]AppRelease, int, error) {
	limit, offset = clampPage(limit, offset)
	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM app_releases
		WHERE product_code = ? AND channel_code = ? AND platform_code = ? AND arch_code = ?
			AND release_status = 'published'`,
		product, channel, platform, arch,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, releaseSelect+`
		WHERE product_code = ? AND channel_code = ? AND platform_code = ? AND arch_code = ?
			AND release_status = 'published'
		ORDER BY published_at DESC
		LIMIT ? OFFSET ?`, product, channel, platform, arch, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanReleaseRows(rows)
	if err != nil {
		return nil, 0, err
	}
	// 按 semver 再排一次（published_at 可能被运营回填不准）
	sortReleasesBySemVerDesc(items)
	return items, total, nil
}

// GetPublishedReleaseByDim 按维度取一条已发布记录；草稿/撤回视为不存在。
func (s *Store) GetPublishedReleaseByDim(ctx context.Context, product, channel, platform, arch, version string) (*AppRelease, error) {
	rel, err := s.GetReleaseByDim(ctx, product, channel, platform, arch, version)
	if err != nil {
		return nil, err
	}
	if rel.Status != "published" {
		return nil, ErrNotFound
	}
	return rel, nil
}

// LatestPublishedRelease 返回 semver 最大的 published 记录。
func (s *Store) LatestPublishedRelease(ctx context.Context, product, channel, platform, arch string) (*AppRelease, error) {
	items, _, err := s.ListPublishedReleases(ctx, product, channel, platform, arch, maxPageSize, 0)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func sortReleasesBySemVerDesc(items []AppRelease) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			cmp, err := CompareSemVerStrings(items[i].Version, items[j].Version)
			if err != nil {
				continue
			}
			if cmp < 0 {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

const releaseSelect = `
	SELECT id, product_code, version_label, channel_code, platform_code, arch_code,
		min_supported_version, force_update, title_text, notes_md, download_url,
		sha256_hex, file_size, release_status, published_at, created_by, created_at, updated_at
	FROM app_releases`

type scannable interface {
	Scan(dest ...any) error
}

func scanRelease(row scannable) (*AppRelease, error) {
	var r AppRelease
	var force int
	var published, created, updated dbTime
	err := row.Scan(
		&r.ID, &r.Product, &r.Version, &r.Channel, &r.Platform, &r.Arch,
		&r.MinSupportedVersion, &force, &r.Title, &r.NotesMd, &r.DownloadURL,
		&r.SHA256, &r.FileSize, &r.Status, &published, &r.CreatedBy, &created, &updated,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	r.ForceUpdate = force != 0
	r.PublishedAt = published.Ptr()
	r.CreatedAt = created.String()
	r.UpdatedAt = updated.String()
	return &r, nil
}

func scanReleaseRows(rows *sql.Rows) ([]AppRelease, error) {
	var out []AppRelease
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
