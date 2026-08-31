package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// AIUpstreamEndpoint 系统模型上游接入点（密钥只存密文）。
type AIUpstreamEndpoint struct {
	EndpointID          string `json:"id"`
	ProviderDisplayName string `json:"providerName"`
	UpstreamBaseURL     string `json:"baseUrl"`
	APISecretCipher     string `json:"-"`
	HasAPISecret        bool   `json:"hasApiSecret"`
	ServiceEnabled      bool   `json:"enabled"`
	DefaultModelCode    string `json:"defaultModel"`
	DailyRequestLimit   int    `json:"dailyRequestLimit"`
	DailyTokenLimit     int    `json:"dailyTokenLimit"`
	SortSeq             int    `json:"sortSeq"`
	CreatedByStaffID    string `json:"createdByStaffId"`
	UpdatedByStaffID    string `json:"updatedByStaffId"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

// AIModelCatalog 接入点下的一条模型目录。
type AIModelCatalog struct {
	CatalogID        string `json:"id"`
	EndpointID       string `json:"endpointId"`
	ModelCode        string `json:"code"`
	ModelLabel       string `json:"label"`
	RecordStatus     string `json:"recordStatus"`
	SortSeq          int    `json:"sortSeq"`
	CreatedByStaffID string `json:"createdByStaffId"`
	UpdatedByStaffID string `json:"updatedByStaffId"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

const aiEndpointColumns = `endpoint_id, provider_display_name, upstream_base_url, api_secret_cipher,
	service_enabled, default_model_code, daily_request_limit, daily_token_limit, sort_seq,
	created_by_staff_id, updated_by_staff_id, created_at, updated_at`

const aiModelColumns = `catalog_id, endpoint_id, model_code, model_label, record_status, sort_seq,
	created_by_staff_id, updated_by_staff_id, created_at, updated_at`

func scanAIEndpoint(row rowScanner) (AIUpstreamEndpoint, error) {
	var (
		e       AIUpstreamEndpoint
		enabled int
		cipher  string
	)
	var createdAt, updatedAt dbTime
	err := row.Scan(
		&e.EndpointID, &e.ProviderDisplayName, &e.UpstreamBaseURL, &cipher,
		&enabled, &e.DefaultModelCode, &e.DailyRequestLimit, &e.DailyTokenLimit, &e.SortSeq,
		&e.CreatedByStaffID, &e.UpdatedByStaffID, &createdAt, &updatedAt,
	)
	if err != nil {
		return e, err
	}
	e.APISecretCipher = cipher
	e.HasAPISecret = strings.TrimSpace(cipher) != ""
	e.ServiceEnabled = enabled != 0
	e.CreatedAt = createdAt.String()
	e.UpdatedAt = updatedAt.String()
	return e, nil
}

func scanAIModel(row rowScanner) (AIModelCatalog, error) {
	var m AIModelCatalog
	var createdAt, updatedAt dbTime
	err := row.Scan(
		&m.CatalogID, &m.EndpointID, &m.ModelCode, &m.ModelLabel, &m.RecordStatus, &m.SortSeq,
		&m.CreatedByStaffID, &m.UpdatedByStaffID, &createdAt, &updatedAt,
	)
	if err != nil {
		return m, err
	}
	m.CreatedAt = createdAt.String()
	m.UpdatedAt = updatedAt.String()
	return m, nil
}

// ListAIEndpoints 列出全部接入点（含关闭），按顺序。
func (s *Store) ListAIEndpoints(ctx context.Context) ([]AIUpstreamEndpoint, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+aiEndpointColumns+` FROM ai_upstream_endpoint
		ORDER BY sort_seq, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIUpstreamEndpoint
	for rows.Next() {
		e, scanErr := scanAIEndpoint(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetAIEndpoint 按 ID 读取接入点；不存在返回 ErrNotFound。
func (s *Store) GetAIEndpoint(ctx context.Context, id string) (*AIUpstreamEndpoint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrNotFound
	}
	e, err := scanAIEndpoint(s.db.QueryRowContext(ctx, `
		SELECT `+aiEndpointColumns+` FROM ai_upstream_endpoint WHERE endpoint_id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetActiveAIEndpoint 取排序最前且已开放的接入点。
func (s *Store) GetActiveAIEndpoint(ctx context.Context) (*AIUpstreamEndpoint, error) {
	e, err := scanAIEndpoint(s.db.QueryRowContext(ctx, `
		SELECT `+aiEndpointColumns+` FROM ai_upstream_endpoint
		WHERE service_enabled = 1
		ORDER BY sort_seq, created_at LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// InsertAIEndpoint 写入接入点。
func (s *Store) InsertAIEndpoint(ctx context.Context, e *AIUpstreamEndpoint) error {
	if e == nil || strings.TrimSpace(e.EndpointID) == "" {
		return errors.New("endpoint id required")
	}
	enabled := 0
	if e.ServiceEnabled {
		enabled = 1
	}
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_upstream_endpoint (
			endpoint_id, provider_display_name, upstream_base_url, api_secret_cipher,
			service_enabled, default_model_code, daily_request_limit, daily_token_limit, sort_seq,
			created_by_staff_id, updated_by_staff_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.EndpointID, e.ProviderDisplayName, e.UpstreamBaseURL, e.APISecretCipher,
		enabled, e.DefaultModelCode, e.DailyRequestLimit, e.DailyTokenLimit, e.SortSeq,
		e.CreatedByStaffID, e.UpdatedByStaffID, now, now,
	)
	return err
}

// UpdateAIEndpoint 更新接入点业务字段与编辑人。
func (s *Store) UpdateAIEndpoint(ctx context.Context, e *AIUpstreamEndpoint) error {
	if e == nil || strings.TrimSpace(e.EndpointID) == "" {
		return ErrNotFound
	}
	enabled := 0
	if e.ServiceEnabled {
		enabled = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE ai_upstream_endpoint SET
			provider_display_name = ?, upstream_base_url = ?, api_secret_cipher = ?,
			service_enabled = ?, default_model_code = ?, daily_request_limit = ?,
			daily_token_limit = ?, sort_seq = ?, updated_by_staff_id = ?, updated_at = ?
		WHERE endpoint_id = ?`,
		e.ProviderDisplayName, e.UpstreamBaseURL, e.APISecretCipher,
		enabled, e.DefaultModelCode, e.DailyRequestLimit, e.DailyTokenLimit, e.SortSeq,
		e.UpdatedByStaffID, nowUTC(), e.EndpointID,
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

// DeleteAIEndpoint 删除接入点并级联其模型目录。
func (s *Store) DeleteAIEndpoint(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrNotFound
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM ai_model_catalog WHERE endpoint_id = ?`, id); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_upstream_endpoint WHERE endpoint_id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAIModels 列出某接入点下的模型；status 空则不过滤。
func (s *Store) ListAIModels(ctx context.Context, endpointID, status string) ([]AIModelCatalog, error) {
	q := `SELECT ` + aiModelColumns + ` FROM ai_model_catalog WHERE endpoint_id = ?`
	args := []any{strings.TrimSpace(endpointID)}
	if status = strings.TrimSpace(status); status != "" {
		q += ` AND record_status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY sort_seq, model_code`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIModelCatalog
	for rows.Next() {
		m, scanErr := scanAIModel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetAIModel 按目录 ID 读取。
func (s *Store) GetAIModel(ctx context.Context, id string) (*AIModelCatalog, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrNotFound
	}
	m, err := scanAIModel(s.db.QueryRowContext(ctx, `
		SELECT `+aiModelColumns+` FROM ai_model_catalog WHERE catalog_id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// InsertAIModel 写入模型目录。
func (s *Store) InsertAIModel(ctx context.Context, m *AIModelCatalog) error {
	if m == nil || strings.TrimSpace(m.CatalogID) == "" {
		return errors.New("catalog id required")
	}
	if m.RecordStatus == "" {
		m.RecordStatus = "active"
	}
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_model_catalog (
			catalog_id, endpoint_id, model_code, model_label, record_status, sort_seq,
			created_by_staff_id, updated_by_staff_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.CatalogID, m.EndpointID, m.ModelCode, m.ModelLabel, m.RecordStatus, m.SortSeq,
		m.CreatedByStaffID, m.UpdatedByStaffID, now, now,
	)
	if err != nil && isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// UpdateAIModel 更新模型目录。
func (s *Store) UpdateAIModel(ctx context.Context, m *AIModelCatalog) error {
	if m == nil || strings.TrimSpace(m.CatalogID) == "" {
		return ErrNotFound
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE ai_model_catalog SET
			model_code = ?, model_label = ?, record_status = ?, sort_seq = ?,
			updated_by_staff_id = ?, updated_at = ?
		WHERE catalog_id = ?`,
		m.ModelCode, m.ModelLabel, m.RecordStatus, m.SortSeq, m.UpdatedByStaffID, nowUTC(), m.CatalogID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAIModel 删除一条模型目录。
func (s *Store) DeleteAIModel(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrNotFound
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_model_catalog WHERE catalog_id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
