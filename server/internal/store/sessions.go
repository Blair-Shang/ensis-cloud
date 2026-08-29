package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type RefreshSession struct {
	ID          string
	UserID      string
	TokenHash   string
	DeviceLabel string
	ExpiresAt   string
	RevokedAt   *string
	CreatedAt   string
}

func (s *Store) CreateRefreshSession(ctx context.Context, sess RefreshSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_sessions (id, user_id, token_hash, device_label, expires_at, revoked_at, created_at)
		VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		sess.ID, sess.UserID, sess.TokenHash, sess.DeviceLabel, toDBTime(sess.ExpiresAt), toDBTime(sess.CreatedAt),
	)
	return err
}

func (s *Store) GetRefreshByHash(ctx context.Context, hash string) (*RefreshSession, error) {
	var sess RefreshSession
	var revoked, expires, created dbTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, device_label, expires_at, revoked_at, created_at
		FROM refresh_sessions WHERE token_hash = ?`, hash,
	).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.DeviceLabel, &expires, &revoked, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.ExpiresAt = expires.String()
	sess.RevokedAt = revoked.Ptr()
	sess.CreatedAt = created.String()
	return &sess, nil
}

func (s *Store) RevokeRefresh(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE refresh_sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		nowUTC(), hash,
	)
	return err
}

func (s *Store) CountActiveSessions(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM refresh_sessions
		WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?`, userID, nowUTC()).Scan(&n)
	return n, err
}

const (
	StaffRoleAdmin    = "admin"
	StaffRoleOperator = "operator"
)

// StaffUser 运营员工。Role 为 admin（超管）或 operator（日常运营）。
type StaffUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	DisplayName  string `json:"displayName"`
	Status       string `json:"status"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// NormalizeStaffRole 只允许 admin / operator；空值按 operator。
func NormalizeStaffRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case StaffRoleAdmin:
		return StaffRoleAdmin
	default:
		return StaffRoleOperator
	}
}

// IsStaffAdmin 判断是否为超管。
func IsStaffAdmin(u *StaffUser) bool {
	return u != nil && u.Role == StaffRoleAdmin
}

func (s *Store) GetStaffByEmail(ctx context.Context, email string) (*StaffUser, error) {
	var u StaffUser
	var created, updated dbTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, display_name, account_status, staff_role, created_at, updated_at
		FROM staff_users WHERE email = ?`, strings.ToLower(email),
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status, &u.Role, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = created.String()
	u.UpdatedAt = updated.String()
	return &u, nil
}

func (s *Store) GetStaffByID(ctx context.Context, id string) (*StaffUser, error) {
	var u StaffUser
	var created, updated dbTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, display_name, account_status, staff_role, created_at, updated_at
		FROM staff_users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status, &u.Role, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = created.String()
	u.UpdatedAt = updated.String()
	return &u, nil
}

func (s *Store) CreateStaff(ctx context.Context, u StaffUser) error {
	updatedAt := u.UpdatedAt
	if updatedAt == "" {
		updatedAt = u.CreatedAt
	}
	role := u.Role
	if role != StaffRoleAdmin && role != StaffRoleOperator {
		role = StaffRoleOperator
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO staff_users (id, email, password_hash, display_name, account_status, staff_role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.DisplayName, u.Status, role, toDBTime(u.CreatedAt), toDBTime(updatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Store) ListStaff(ctx context.Context, q string, limit, offset int) ([]StaffUser, int, error) {
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
		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM staff_users`).Scan(&total)
	} else {
		like := "%" + q + "%"
		err = s.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM staff_users WHERE email LIKE ? OR display_name LIKE ?`, like, like).Scan(&total)
	}
	if err != nil {
		return nil, 0, err
	}

	var rows *sql.Rows
	if q == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, email, password_hash, display_name, account_status, staff_role, created_at, updated_at
			FROM staff_users
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?`, limit, offset)
	} else {
		like := "%" + q + "%"
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, email, password_hash, display_name, account_status, staff_role, created_at, updated_at
			FROM staff_users
			WHERE email LIKE ? OR display_name LIKE ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?`, like, like, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]StaffUser, 0, limit)
	for rows.Next() {
		var u StaffUser
		var created, updated dbTime
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Status, &u.Role, &created, &updated); err != nil {
			return nil, 0, err
		}
		u.CreatedAt = created.String()
		u.UpdatedAt = updated.String()
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (s *Store) UpdateStaffPassword(ctx context.Context, id, hash string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE staff_users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateStaffProfile(ctx context.Context, id, displayName, status, role string) error {
	role = NormalizeStaffRole(role)
	res, err := s.db.ExecContext(ctx, `
		UPDATE staff_users SET display_name = ?, account_status = ?, staff_role = ?, updated_at = ? WHERE id = ?`,
		displayName, status, role, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func PublicStaff(u *StaffUser) map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":          u.ID,
		"email":       u.Email,
		"displayName": u.DisplayName,
		"status":      u.Status,
		"role":        NormalizeStaffRole(u.Role),
		"createdAt":   u.CreatedAt,
		"updatedAt":   u.UpdatedAt,
	}
}

// CountActiveAdmins 统计未停用的超管人数。
func (s *Store) CountActiveAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM staff_users
		WHERE staff_role = ? AND account_status = ?`, StaffRoleAdmin, "active").Scan(&n)
	return n, err
}

type StaffSession struct {
	ID        string
	StaffID   string
	TokenHash string
	ExpiresAt string
	RevokedAt *string
	CreatedAt string
}

func (s *Store) CreateStaffSession(ctx context.Context, id, staffID, tokenHash, expiresAt, createdAt string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO staff_sessions (id, staff_id, token_hash, expires_at, revoked_at, created_at)
		VALUES (?, ?, ?, ?, NULL, ?)`, id, staffID, tokenHash, toDBTime(expiresAt), toDBTime(createdAt))
	return err
}

func (s *Store) GetStaffSessionByHash(ctx context.Context, hash string) (*StaffSession, error) {
	var sess StaffSession
	var revoked, expires, created dbTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, staff_id, token_hash, expires_at, revoked_at, created_at
		FROM staff_sessions WHERE token_hash = ?`, hash,
	).Scan(&sess.ID, &sess.StaffID, &sess.TokenHash, &expires, &revoked, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.ExpiresAt = expires.String()
	sess.RevokedAt = revoked.Ptr()
	sess.CreatedAt = created.String()
	return &sess, nil
}

func (s *Store) RevokeStaffSession(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE staff_sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		nowUTC(), hash,
	)
	return err
}

// RevokeStaffSessionsByStaffID 吊销某员工全部未过期会话（改密 / 重置后强制重登）。
func (s *Store) RevokeStaffSessionsByStaffID(ctx context.Context, staffID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE staff_sessions SET revoked_at = ? WHERE staff_id = ? AND revoked_at IS NULL`,
		nowUTC(), staffID,
	)
	return err
}

// RevokeRefreshByUserID 吊销某终端用户全部 refresh 会话。
func (s *Store) RevokeRefreshByUserID(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE refresh_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		nowUTC(), userID,
	)
	return err
}
