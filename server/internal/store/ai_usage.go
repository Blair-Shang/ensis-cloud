package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AIUsage 是某用户某日的系统模型用量。
type AIUsage struct {
	RequestCount int
	TokenCount   int
}

// AIUsageRow 是运营按日用量列表的一行；用户已删时邮箱与显示名为空。
type AIUsageRow struct {
	UserID        string `json:"userId"`
	Email         string `json:"email"`
	DisplayName   string `json:"displayName"`
	AccountStatus string `json:"status"`
	UsageDay      string `json:"usageDay"`
	RequestCount  int    `json:"requestCount"`
	TokenCount    int    `json:"tokenCount"`
	UpdatedAt     string `json:"updatedAt"`
}

// AIUsageSummary 是某 UTC 日有实际调用的用户数与次数汇总。
type AIUsageSummary struct {
	UsageDay     string `json:"usageDay"`
	UserCount    int    `json:"userCount"`
	RequestCount int    `json:"requestCount"`
	TokenCount   int    `json:"tokenCount"`
}

// AIUsageListFilter 运营用量列表：按 UTC 日等值筛选，禁止模糊扫全表。
type AIUsageListFilter struct {
	Day    string
	UserID string
	Email  string
	Limit  int
	Offset int
}

func usageDayUTC(t time.Time) string {
	if t.IsZero() {
		t = nowUTC()
	}
	return t.UTC().Format("2006-01-02")
}

// ParseUsageDay 解析 UTC 日 YYYY-MM-DD；空则用 fallback 的 UTC 日。
func ParseUsageDay(raw string, fallback time.Time) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return usageDayUTC(fallback), nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return "", fmt.Errorf("invalid usage day")
	}
	return t.Format("2006-01-02"), nil
}

// GetAIUsage 读取用户当日用量；无记录时返回零值。
func (s *Store) GetAIUsage(ctx context.Context, userID string, at time.Time) (AIUsage, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AIUsage{}, errors.New("user id required")
	}
	var out AIUsage
	err := s.db.QueryRowContext(ctx, `
		SELECT request_count, token_count FROM ai_usage_daily
		WHERE user_id = ? AND usage_day = ?`, userID, usageDayUTC(at),
	).Scan(&out.RequestCount, &out.TokenCount)
	if errors.Is(err, sql.ErrNoRows) {
		return AIUsage{}, nil
	}
	return out, err
}

// AddAIUsage 累加当日请求与 token，返回更新后的用量。
func (s *Store) AddAIUsage(ctx context.Context, userID string, requests, tokens int, at time.Time) (AIUsage, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AIUsage{}, errors.New("user id required")
	}
	if requests < 0 {
		requests = 0
	}
	if tokens < 0 {
		tokens = 0
	}
	day := usageDayUTC(at)
	now := nowUTC()
	var q string
	if s.driver == "mysql" {
		q = `
			INSERT INTO ai_usage_daily (
				user_id, usage_day, request_count, token_count,
				created_by_staff_id, updated_by_staff_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, '', '', ?, ?)
			ON DUPLICATE KEY UPDATE
				request_count = request_count + VALUES(request_count),
				token_count = token_count + VALUES(token_count),
				updated_at = VALUES(updated_at)`
	} else {
		q = `
			INSERT INTO ai_usage_daily (
				user_id, usage_day, request_count, token_count,
				created_by_staff_id, updated_by_staff_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, '', '', ?, ?)
			ON CONFLICT(user_id, usage_day) DO UPDATE SET
				request_count = request_count + excluded.request_count,
				token_count = token_count + excluded.token_count,
				updated_at = excluded.updated_at`
	}
	if _, err := s.db.ExecContext(ctx, q, userID, day, requests, tokens, now, now); err != nil {
		return AIUsage{}, err
	}
	return s.GetAIUsage(ctx, userID, at)
}

// TryAddAIUsage 在限额内原子累加；超限返回 ErrQuotaExceeded 且不增加计数。
// requestLimit / tokenLimit <=0 表示该项不限制。
func (s *Store) TryAddAIUsage(ctx context.Context, userID string, requests, tokens, requestLimit, tokenLimit int, at time.Time) (AIUsage, error) {
	if _, err := s.AddAIUsage(ctx, userID, 0, 0, at); err != nil {
		return AIUsage{}, err
	}
	userID = strings.TrimSpace(userID)
	if requests < 0 {
		requests = 0
	}
	if tokens < 0 {
		tokens = 0
	}
	day := usageDayUTC(at)
	now := nowUTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE ai_usage_daily
		SET request_count = request_count + ?,
			token_count = token_count + ?,
			updated_at = ?
		WHERE user_id = ? AND usage_day = ?
			AND (? <= 0 OR request_count + ? <= ?)
			AND (? <= 0 OR token_count + ? <= ?)`,
		requests, tokens, now, userID, day,
		requestLimit, requests, requestLimit,
		tokenLimit, tokens, tokenLimit,
	)
	if err != nil {
		return AIUsage{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return AIUsage{}, ErrQuotaExceeded
	}
	return s.GetAIUsage(ctx, userID, at)
}

// SummarizeAIUsage 汇总某 UTC 日 request_count>0 的用户数与次数。
func (s *Store) SummarizeAIUsage(ctx context.Context, day string) (AIUsageSummary, error) {
	day, err := ParseUsageDay(day, nowUTC())
	if err != nil {
		return AIUsageSummary{}, err
	}
	out := AIUsageSummary{UsageDay: day}
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(request_count), 0), COALESCE(SUM(token_count), 0)
		FROM ai_usage_daily
		WHERE usage_day = ? AND request_count > 0`, day,
	).Scan(&out.UserCount, &out.RequestCount, &out.TokenCount)
	return out, err
}

// ListAIUsageTrend 返回含 end 日在内最近 days 天的按日汇总，缺日补零。
func (s *Store) ListAIUsageTrend(ctx context.Context, end string, days int) ([]AIUsageSummary, error) {
	end, err := ParseUsageDay(end, nowUTC())
	if err != nil {
		return nil, err
	}
	if days <= 0 {
		days = 7
	}
	if days > 31 {
		days = 31
	}
	endT, err := time.Parse("2006-01-02", end)
	if err != nil {
		return nil, err
	}
	startT := endT.AddDate(0, 0, -(days - 1))
	start := startT.Format("2006-01-02")

	rows, err := s.db.QueryContext(ctx, `
		SELECT usage_day, COUNT(*), COALESCE(SUM(request_count), 0), COALESCE(SUM(token_count), 0)
		FROM ai_usage_daily
		WHERE usage_day >= ? AND usage_day <= ? AND request_count > 0
		GROUP BY usage_day`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDay := make(map[string]AIUsageSummary, days)
	for rows.Next() {
		var (
			dayVal any
			row    AIUsageSummary
		)
		if err := rows.Scan(&dayVal, &row.UserCount, &row.RequestCount, &row.TokenCount); err != nil {
			return nil, err
		}
		row.UsageDay = formatUsageDay(dayVal)
		byDay[row.UsageDay] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]AIUsageSummary, 0, days)
	for i := 0; i < days; i++ {
		d := startT.AddDate(0, 0, i).Format("2006-01-02")
		if row, ok := byDay[d]; ok {
			out = append(out, row)
			continue
		}
		out = append(out, AIUsageSummary{UsageDay: d})
	}
	return out, nil
}

// ListAIUsage 按 UTC 日列出用户用量，请求次数降序。
func (s *Store) ListAIUsage(ctx context.Context, f AIUsageListFilter) ([]AIUsageRow, int, error) {
	day, err := ParseUsageDay(f.Day, nowUTC())
	if err != nil {
		return nil, 0, err
	}
	limit, offset := clampPage(f.Limit, f.Offset)
	where := []string{"u.usage_day = ?", "u.request_count > 0"}
	args := []any{day}
	appendEq(&where, &args, "u.user_id", f.UserID)
	if email := strings.ToLower(strings.TrimSpace(f.Email)); email != "" {
		where = append(where, "usr.email = ?")
		args = append(args, email)
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM ai_usage_daily u
		LEFT JOIN users usr ON usr.id = u.user_id
		WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.user_id, usr.email, usr.display_name, usr.account_status,
			u.usage_day, u.request_count, u.token_count, u.updated_at
		FROM ai_usage_daily u
		LEFT JOIN users usr ON usr.id = u.user_id
		WHERE `+clause+`
		ORDER BY u.request_count DESC, u.updated_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AIUsageRow
	for rows.Next() {
		var (
			row                 AIUsageRow
			email, name, status sql.NullString
			dayVal              any
			updated             dbTime
		)
		if err := rows.Scan(
			&row.UserID, &email, &name, &status,
			&dayVal, &row.RequestCount, &row.TokenCount, &updated,
		); err != nil {
			return nil, 0, err
		}
		row.Email = strings.TrimSpace(email.String)
		row.DisplayName = strings.TrimSpace(name.String)
		row.AccountStatus = strings.TrimSpace(status.String)
		row.UsageDay = formatUsageDay(dayVal)
		row.UpdatedAt = updated.String()
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func formatUsageDay(src any) string {
	switch v := src.(type) {
	case nil:
		return ""
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.UTC().Format("2006-01-02")
	case []byte:
		return formatUsageDay(string(v))
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return ""
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t.Format("2006-01-02")
		}
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
