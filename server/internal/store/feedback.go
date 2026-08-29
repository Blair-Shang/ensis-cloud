package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// FeedbackAttachment 反馈截图元数据（实体文件在 data_dir/feedback/{id}/）。
type FeedbackAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	File        string `json:"file"`
}

// Feedback JSON 字段保持对外固定契约（title/body/status…），映射到专业库列。
type Feedback struct {
	ID              string               `json:"id"`
	UserID          string               `json:"userId,omitempty"`
	Category        string               `json:"category"`
	Title           string               `json:"title"`
	Body            string               `json:"body"`
	Contact         string               `json:"contact,omitempty"`
	Product         string               `json:"product,omitempty"`
	ClientVersion   string               `json:"clientVersion,omitempty"`
	Status          string               `json:"status"`
	AdminNote       string               `json:"adminNote,omitempty"`
	StaffReply      string               `json:"staffReply,omitempty"`
	StaffReplyAt    *string              `json:"staffReplyAt,omitempty"`
	StaffReplierID  string               `json:"staffReplierId,omitempty"`
	ReplyNotifiedAt *string              `json:"replyNotifiedAt,omitempty"`
	Attachments     []FeedbackAttachment `json:"attachments,omitempty"`
	IPHash          string               `json:"-"`
	CreatedAt       string               `json:"createdAt"`
	UpdatedAt       string               `json:"updatedAt"`
}

type FeedbackInput struct {
	ID            string
	UserID        string
	Category      string
	Title         string
	Body          string
	Contact       string
	Product       string
	ClientVersion string
	IPHash        string
	Attachments   []FeedbackAttachment
}

type FeedbackUpdate struct {
	Status         string
	AdminNote      string
	StaffReply     string
	StaffReplierID string
	TouchReply     bool // staff_reply 是否本次变更
}

func (s *Store) InsertFeedback(ctx context.Context, in FeedbackInput) (*Feedback, error) {
	now := nowUTC()
	atts := in.Attachments
	if atts == nil {
		atts = []FeedbackAttachment{}
	}
	attsJSON, err := json.Marshal(atts)
	if err != nil {
		return nil, err
	}
	f := &Feedback{
		ID:            in.ID,
		UserID:        in.UserID,
		Category:      in.Category,
		Title:         in.Title,
		Body:          in.Body,
		Contact:       in.Contact,
		Product:       in.Product,
		ClientVersion: in.ClientVersion,
		Status:        "open",
		Attachments:   atts,
		IPHash:        in.IPHash,
		CreatedAt:     formatTime(now),
		UpdatedAt:     formatTime(now),
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO feedback (
			id, reporter_user_id, category_code, subject, content_text, contact_email, product_code, client_version,
			ticket_status, internal_note, staff_reply, staff_reply_at, staff_replier_id, reply_notified_at,
			source_ip_hash, attachments_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', NULL, '', NULL, ?, ?, ?, ?)`,
		f.ID, f.UserID, f.Category, f.Title, f.Body, f.Contact, f.Product, f.ClientVersion,
		f.Status, f.IPHash, string(attsJSON), now, now,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) ListFeedbackByUser(ctx context.Context, userID string, limit int) ([]Feedback, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, reporter_user_id, category_code, subject, content_text, contact_email, product_code, client_version,
			ticket_status, internal_note, staff_reply, staff_reply_at, staff_replier_id, reply_notified_at,
			source_ip_hash, COALESCE(attachments_json, '[]'), created_at, updated_at
		FROM feedback WHERE reporter_user_id = ? ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFeedbackRows(rows)
}

// FeedbackListFilter 工单列表：字段等值，正文不再 LIKE 全表扫描。
type FeedbackListFilter struct {
	ID            string
	Title         string
	Contact       string
	Category      string
	Status        string
	Product       string
	UserID        string
	ClientVersion string
	Limit         int
	Offset        int
}

func (s *Store) ListFeedback(ctx context.Context, f FeedbackListFilter) ([]Feedback, int, error) {
	limit, offset := clampPage(f.Limit, f.Offset)
	where := []string{"1=1"}
	args := []any{}
	appendEq(&where, &args, "id", f.ID)
	appendEq(&where, &args, "subject", f.Title)
	if contact := strings.ToLower(strings.TrimSpace(f.Contact)); contact != "" {
		where = append(where, "LOWER(contact_email) = ?")
		args = append(args, contact)
	}
	appendEq(&where, &args, "category_code", f.Category)
	appendEq(&where, &args, "ticket_status", f.Status)
	appendEq(&where, &args, "product_code", f.Product)
	appendEq(&where, &args, "reporter_user_id", f.UserID)
	appendEq(&where, &args, "client_version", f.ClientVersion)
	whereSQL := strings.Join(where, " AND ")

	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feedback WHERE `+whereSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, reporter_user_id, category_code, subject, content_text, contact_email, product_code, client_version,
			ticket_status, internal_note, staff_reply, staff_reply_at, staff_replier_id, reply_notified_at,
			source_ip_hash, COALESCE(attachments_json, '[]'), created_at, updated_at
		FROM feedback WHERE `+whereSQL+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanFeedbackRows(rows)
	return items, total, err
}

func (s *Store) GetFeedback(ctx context.Context, id string) (*Feedback, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, reporter_user_id, category_code, subject, content_text, contact_email, product_code, client_version,
			ticket_status, internal_note, staff_reply, staff_reply_at, staff_replier_id, reply_notified_at,
			source_ip_hash, COALESCE(attachments_json, '[]'), created_at, updated_at
		FROM feedback WHERE id = ?`, id)
	f, err := scanFeedbackRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

// GetFeedbackByIDAndContact 官网匿名查询：工单 ID + 联系邮箱必须同时匹配。
func (s *Store) GetFeedbackByIDAndContact(ctx context.Context, id, contact string) (*Feedback, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, reporter_user_id, category_code, subject, content_text, contact_email, product_code, client_version,
			ticket_status, internal_note, staff_reply, staff_reply_at, staff_replier_id, reply_notified_at,
			source_ip_hash, COALESCE(attachments_json, '[]'), created_at, updated_at
		FROM feedback WHERE id = ? AND LOWER(contact_email) = ?`,
		id, strings.ToLower(strings.TrimSpace(contact)),
	)
	f, err := scanFeedbackRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *Store) UpdateFeedback(ctx context.Context, id string, upd FeedbackUpdate) (*Feedback, error) {
	f, err := s.GetFeedback(ctx, id)
	if err != nil {
		return nil, err
	}
	if upd.Status != "" {
		f.Status = upd.Status
	}
	f.AdminNote = upd.AdminNote
	now := nowUTC()
	f.UpdatedAt = formatTime(now)

	var replyAt any
	if upd.TouchReply {
		f.StaffReply = upd.StaffReply
		f.StaffReplierID = upd.StaffReplierID
		if strings.TrimSpace(upd.StaffReply) != "" {
			replyAt = now
			s := formatTime(now)
			f.StaffReplyAt = &s
		} else {
			replyAt = nil
			f.StaffReplyAt = nil
			f.StaffReplierID = ""
		}
	} else {
		replyAt = toDBNullTime(f.StaffReplyAt)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE feedback SET
			ticket_status = ?, internal_note = ?, staff_reply = ?, staff_reply_at = ?,
			staff_replier_id = ?, updated_at = ?
		WHERE id = ?`,
		f.Status, f.AdminNote, f.StaffReply, replyAt, f.StaffReplierID, now, id,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) MarkFeedbackReplyNotified(ctx context.Context, id string) error {
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE feedback SET reply_notified_at = ?, updated_at = ? WHERE id = ?`,
		now, now, id,
	)
	return err
}

func (s *Store) CountFeedbackSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM feedback WHERE created_at >= ?`, since.UTC(),
	).Scan(&n)
	return n, err
}

func (s *Store) CountFeedback(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feedback`).Scan(&n)
	return n, err
}

// PublicFeedback 用户侧可见字段（隐藏内部备注）。
func PublicFeedback(f Feedback) map[string]any {
	atts := f.Attachments
	if atts == nil {
		atts = []FeedbackAttachment{}
	}
	return map[string]any{
		"id":            f.ID,
		"userId":        f.UserID,
		"category":      f.Category,
		"title":         f.Title,
		"body":          f.Body,
		"contact":       f.Contact,
		"product":       f.Product,
		"clientVersion": f.ClientVersion,
		"status":        f.Status,
		"staffReply":    f.StaffReply,
		"staffReplyAt":  f.StaffReplyAt,
		"attachments":   atts,
		"createdAt":     f.CreatedAt,
		"updatedAt":     f.UpdatedAt,
	}
}

func scanFeedbackRows(rows *sql.Rows) ([]Feedback, error) {
	var out []Feedback
	for rows.Next() {
		f, err := scanFeedback(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFeedbackRow(row *sql.Row) (*Feedback, error) {
	return scanFeedback(row)
}

func scanFeedback(row rowScanner) (*Feedback, error) {
	var f Feedback
	var replyAt, notifiedAt, created, updated dbTime
	var attsJSON string
	err := row.Scan(
		&f.ID, &f.UserID, &f.Category, &f.Title, &f.Body, &f.Contact, &f.Product, &f.ClientVersion,
		&f.Status, &f.AdminNote, &f.StaffReply, &replyAt, &f.StaffReplierID, &notifiedAt,
		&f.IPHash, &attsJSON, &created, &updated,
	)
	if err != nil {
		return nil, err
	}
	f.StaffReplyAt = replyAt.Ptr()
	f.ReplyNotifiedAt = notifiedAt.Ptr()
	f.CreatedAt = created.String()
	f.UpdatedAt = updated.String()
	if strings.TrimSpace(attsJSON) != "" && attsJSON != "null" {
		_ = json.Unmarshal([]byte(attsJSON), &f.Attachments)
	}
	if f.Attachments == nil {
		f.Attachments = []FeedbackAttachment{}
	}
	return &f, nil
}
