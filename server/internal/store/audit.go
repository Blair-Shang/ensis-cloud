package store

import (
	"context"
	"encoding/json"
	"strings"
)

type AuditLog struct {
	ID           string
	ActorKind    string // user | staff | system
	ActorID      string
	ActionCode   string
	ResourceKind string
	ResourceID   string
	Detail       any
	SourceIPHash string
}

// AuditLogItem 审计列表对外结构（detail 已解析为 JSON 对象）。
type AuditLogItem struct {
	ID           string          `json:"id"`
	ActorKind    string          `json:"actorKind"`
	ActorID      string          `json:"actorID"`
	ActionCode   string          `json:"actionCode"`
	ResourceKind string          `json:"resourceKind"`
	ResourceID   string          `json:"resourceID"`
	Detail       json.RawMessage `json:"detail"`
	SourceIPHash string          `json:"sourceIPHash"`
	CreatedAt    string          `json:"createdAt"`
}

type AuditListFilter struct {
	ActionCode string
	ActorKind  string
	Q          string // 模糊匹配 action / resource_id / actor_id
	Limit      int
	Offset     int
}

func (s *Store) InsertAuditLog(ctx context.Context, a AuditLog) error {
	detail := "{}"
	if a.Detail != nil {
		b, err := json.Marshal(a.Detail)
		if err != nil {
			return err
		}
		detail = string(b)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, actor_kind, actor_id, action_code, resource_kind, resource_id, detail_json, source_ip_hash, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ActorKind, a.ActorID, a.ActionCode, a.ResourceKind, a.ResourceID, detail, a.SourceIPHash, nowUTC(),
	)
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, f AuditListFilter) ([]AuditLogItem, int, error) {
	limit, offset := f.Limit, f.Offset
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	where := []string{"1=1"}
	args := make([]any, 0, 8)
	if action := strings.TrimSpace(f.ActionCode); action != "" {
		where = append(where, "action_code = ?")
		args = append(args, action)
	}
	if kind := strings.TrimSpace(f.ActorKind); kind != "" {
		where = append(where, "actor_kind = ?")
		args = append(args, kind)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + q + "%"
		where = append(where, "(action_code LIKE ? OR resource_id LIKE ? OR actor_id LIKE ? OR resource_kind LIKE ?)")
		args = append(args, like, like, like, like)
	}
	clause := strings.Join(where, " AND ")

	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE `+clause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_kind, actor_id, action_code, resource_kind, resource_id, detail_json, source_ip_hash, created_at
		FROM audit_logs
		WHERE `+clause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]AuditLogItem, 0, limit)
	for rows.Next() {
		var it AuditLogItem
		var detail string
		var created dbTime
		if err := rows.Scan(
			&it.ID, &it.ActorKind, &it.ActorID, &it.ActionCode, &it.ResourceKind, &it.ResourceID,
			&detail, &it.SourceIPHash, &created,
		); err != nil {
			return nil, 0, err
		}
		if strings.TrimSpace(detail) == "" {
			detail = "{}"
		}
		it.Detail = json.RawMessage(detail)
		it.CreatedAt = created.String()
		out = append(out, it)
	}
	return out, total, rows.Err()
}
