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
	ActionCode   string
	ActorKind    string
	ActorID      string
	ResourceKind string
	ResourceID   string
	Limit        int
	Offset       int
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
	limit, offset := clampPage(f.Limit, f.Offset)

	where := []string{"1=1"}
	args := make([]any, 0, 8)
	appendEq(&where, &args, "action_code", f.ActionCode)
	appendEq(&where, &args, "actor_kind", f.ActorKind)
	appendEq(&where, &args, "actor_id", f.ActorID)
	appendEq(&where, &args, "resource_kind", f.ResourceKind)
	appendEq(&where, &args, "resource_id", f.ResourceID)
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
