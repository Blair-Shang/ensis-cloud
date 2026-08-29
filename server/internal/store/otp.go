package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type OTP struct {
	ID         string
	Email      string
	Purpose    string
	CodeHash   string
	ExpiresAt  string
	ConsumedAt *string
	CreatedAt  string
}

func (s *Store) InsertOTP(ctx context.Context, o OTP) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_otps (id, email, purpose_code, code_hash, expires_at, consumed_at, created_at)
		VALUES (?, ?, ?, ?, ?, NULL, ?)`,
		o.ID, strings.ToLower(o.Email), o.Purpose, o.CodeHash, toDBTime(o.ExpiresAt), toDBTime(o.CreatedAt),
	)
	return err
}

func (s *Store) CountOTPSince(ctx context.Context, email, purpose string, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auth_otps
		WHERE email = ? AND purpose_code = ? AND created_at >= ?`,
		strings.ToLower(email), purpose, since.UTC(),
	).Scan(&n)
	return n, err
}

func (s *Store) LatestOTPCreatedAt(ctx context.Context, email, purpose string) (*time.Time, error) {
	var created dbTime
	err := s.db.QueryRowContext(ctx, `
		SELECT created_at FROM auth_otps
		WHERE email = ? AND purpose_code = ?
		ORDER BY created_at DESC LIMIT 1`,
		strings.ToLower(email), purpose,
	).Scan(&created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !created.Valid {
		return nil, nil
	}
	t := created.Time
	return &t, nil
}

func (s *Store) ConsumeOTP(ctx context.Context, email, purpose, codeHash string) error {
	now := nowUTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE auth_otps SET consumed_at = ?
		WHERE id = (
			SELECT id FROM (
				SELECT id FROM auth_otps
				WHERE email = ? AND purpose_code = ? AND code_hash = ?
				  AND consumed_at IS NULL AND expires_at > ?
				ORDER BY created_at DESC LIMIT 1
			) t
		)`,
		now, strings.ToLower(email), purpose, codeHash, now,
	)
	if err != nil {
		return s.consumeOTPFallback(ctx, email, purpose, codeHash, now)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return s.consumeOTPFallback(ctx, email, purpose, codeHash, now)
	}
	return nil
}

func (s *Store) consumeOTPFallback(ctx context.Context, email, purpose, codeHash string, now time.Time) error {
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM auth_otps
		WHERE email = ? AND purpose_code = ? AND code_hash = ?
		  AND consumed_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		strings.ToLower(email), purpose, codeHash, now,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE auth_otps SET consumed_at = ? WHERE id = ? AND consumed_at IS NULL`, now, id)
	return err
}
