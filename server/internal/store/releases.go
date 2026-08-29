package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

// PublicRelease 公开面：不含 createdBy 等内部字段。
func PublicRelease(r *AppRelease) map[string]any {
	if r == nil {
		return nil
	}
	out := map[string]any{
		"product":             r.Product,
		"channel":             r.Channel,
		"platform":            r.Platform,
		"arch":                r.Arch,
		"version":             r.Version,
		"title":               r.Title,
		"notesMd":             r.NotesMd,
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

type ReleaseListFilter struct {
	Product  string
	Channel  string
	Platform string
	Arch     string
	Status   string
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
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	where := []string{"1=1"}
	args := []any{}
	if p := strings.TrimSpace(f.Product); p != "" {
		where = append(where, "product_code = ?")
		args = append(args, p)
	}
	if c := strings.TrimSpace(f.Channel); c != "" {
		where = append(where, "channel_code = ?")
		args = append(args, c)
	}
	if p := strings.TrimSpace(f.Platform); p != "" {
		where = append(where, "platform_code = ?")
		args = append(args, p)
	}
	if a := strings.TrimSpace(f.Arch); a != "" {
		where = append(where, "arch_code = ?")
		args = append(args, a)
	}
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

// ListPublishedReleases 仅 published，供公开 changelog。
func (s *Store) ListPublishedReleases(ctx context.Context, product, channel, platform, arch string, limit int) ([]AppRelease, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, releaseSelect+`
		WHERE product_code = ? AND channel_code = ? AND platform_code = ? AND arch_code = ?
			AND release_status = 'published'
		ORDER BY published_at DESC
		LIMIT ?`, product, channel, platform, arch, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanReleaseRows(rows)
	if err != nil {
		return nil, err
	}
	// 按 semver 再排一次（published_at 可能被运营回填不准）
	sortReleasesBySemVerDesc(items)
	return items, nil
}

// LatestPublishedRelease 返回 semver 最大的 published 记录。
func (s *Store) LatestPublishedRelease(ctx context.Context, product, channel, platform, arch string) (*AppRelease, error) {
	items, err := s.ListPublishedReleases(ctx, product, channel, platform, arch, 50)
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
