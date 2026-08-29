package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type User struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	PasswordHash    string  `json:"-"`
	DisplayName     string  `json:"displayName"`
	EmailVerifiedAt *string `json:"emailVerifiedAt,omitempty"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

func (s *Store) CreateUser(ctx context.Context, u User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, email_verified_at, account_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.DisplayName, toDBNullTime(u.EmailVerifiedAt),
		u.Status, toDBTime(u.CreatedAt), toDBTime(u.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, display_name, email_verified_at, account_status, created_at, updated_at
		FROM users WHERE email = ?`, strings.ToLower(email)))
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, display_name, email_verified_at, account_status, created_at, updated_at
		FROM users WHERE id = ?`, id))
}

func (s *Store) UpdateUserDisplayName(ctx context.Context, id, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?`, name, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserPassword(ctx context.Context, id, hash string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET account_status = ?, updated_at = ? WHERE id = ?`, status, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type UserListItem struct {
	User
	FeedbackCount int `json:"feedbackCount"`
	SessionCount  int `json:"sessionCount"`
}

func (s *Store) ListUsers(ctx context.Context, q string, limit, offset int) ([]UserListItem, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q = strings.TrimSpace(q)
	var total int
	var err error
	if q == "" {
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	} else {
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE email LIKE ? OR display_name LIKE ?`, "%"+q+"%", "%"+q+"%").Scan(&total)
	}
	if err != nil {
		return nil, 0, err
	}

	now := nowUTC()
	var rows *sql.Rows
	if q == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT u.id, u.email, u.password_hash, u.display_name, u.email_verified_at, u.account_status, u.created_at, u.updated_at,
				(SELECT COUNT(*) FROM feedback f WHERE f.reporter_user_id = u.id) AS feedback_count,
				(SELECT COUNT(*) FROM refresh_sessions r WHERE r.user_id = u.id AND r.revoked_at IS NULL AND r.expires_at > ?) AS session_count
			FROM users u
			ORDER BY u.created_at DESC
			LIMIT ? OFFSET ?`, now, limit, offset)
	} else {
		like := "%" + q + "%"
		rows, err = s.db.QueryContext(ctx, `
			SELECT u.id, u.email, u.password_hash, u.display_name, u.email_verified_at, u.account_status, u.created_at, u.updated_at,
				(SELECT COUNT(*) FROM feedback f WHERE f.reporter_user_id = u.id) AS feedback_count,
				(SELECT COUNT(*) FROM refresh_sessions r WHERE r.user_id = u.id AND r.revoked_at IS NULL AND r.expires_at > ?) AS session_count
			FROM users u
			WHERE u.email LIKE ? OR u.display_name LIKE ?
			ORDER BY u.created_at DESC
			LIMIT ? OFFSET ?`, now, like, like, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []UserListItem
	for rows.Next() {
		var item UserListItem
		var verified, created, updated dbTime
		if err := rows.Scan(
			&item.ID, &item.Email, &item.PasswordHash, &item.DisplayName, &verified,
			&item.Status, &created, &updated, &item.FeedbackCount, &item.SessionCount,
		); err != nil {
			return nil, 0, err
		}
		item.EmailVerifiedAt = verified.Ptr()
		item.CreatedAt = created.String()
		item.UpdatedAt = updated.String()
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	var verified, created, updated dbTime
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &verified, &u.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.EmailVerifiedAt = verified.Ptr()
	u.CreatedAt = created.String()
	u.UpdatedAt = updated.String()
	return &u, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

func PublicUser(u *User) map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":              u.ID,
		"email":           u.Email,
		"displayName":     u.DisplayName,
		"emailVerifiedAt": u.EmailVerifiedAt,
		"status":          u.Status,
		"createdAt":       u.CreatedAt,
	}
}

func EnsureEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") || len(email) > 320 {
		return fmt.Errorf("invalid email")
	}
	return nil
}
