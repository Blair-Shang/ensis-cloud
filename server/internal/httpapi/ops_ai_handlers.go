package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/ensis-cloud/server/internal/auth"
	"github.com/Blair-Shang/ensis-cloud/server/internal/llm"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

type opsAIEndpointCreateReq struct {
	ProviderName      string `json:"providerName"`
	BaseURL           string `json:"baseUrl"`
	APISecret         string `json:"apiSecret"`
	Enabled           *bool  `json:"enabled"`
	DefaultModel      string `json:"defaultModel"`
	DailyRequestLimit *int   `json:"dailyRequestLimit"`
	DailyTokenLimit   *int   `json:"dailyTokenLimit"`
	SortSeq           *int   `json:"sortSeq"`
}

type opsAIEndpointPatchReq struct {
	ProviderName      *string `json:"providerName"`
	BaseURL           *string `json:"baseUrl"`
	APISecret         *string `json:"apiSecret"`
	Enabled           *bool   `json:"enabled"`
	DefaultModel      *string `json:"defaultModel"`
	DailyRequestLimit *int    `json:"dailyRequestLimit"`
	DailyTokenLimit   *int    `json:"dailyTokenLimit"`
	SortSeq           *int    `json:"sortSeq"`
}

type opsAIModelWriteReq struct {
	Code         string `json:"code"`
	Label        string `json:"label"`
	RecordStatus string `json:"recordStatus"`
	SortSeq      *int   `json:"sortSeq"`
}

type opsAIEndpointTestReq struct {
	EndpointID string `json:"endpointId"`
	BaseURL    string `json:"baseUrl"`
	APISecret  string `json:"apiSecret"`
	Model      string `json:"model"`
}

func (s *Server) handleOpsAIEndpointList(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	items, err := s.store.ListAIEndpoints(r.Context())
	if err != nil {
		s.log.Error("list ai endpoints", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.AIUpstreamEndpoint{}
	}
	out := make([]store.AIUpstreamEndpoint, 0, len(items))
	for _, e := range items {
		e.APISecretCipher = ""
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleOpsAIEndpointCreate(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	var req opsAIEndpointCreateReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	name := strings.TrimSpace(req.ProviderName)
	base := strings.TrimSpace(req.BaseURL)
	if name == "" || base == "" {
		writeError(w, http.StatusBadRequest, "invalid_endpoint")
		return
	}
	if err := llm.ValidateUpstreamURL(base); err != nil {
		writeError(w, http.StatusBadRequest, "unsafe_upstream")
		return
	}
	cipher := ""
	if secret := strings.TrimSpace(req.APISecret); secret != "" {
		c, err := s.sealSecret(secret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error")
			return
		}
		cipher = c
	}
	id, err := auth.NewID("aiend")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	limit := 80
	if req.DailyRequestLimit != nil {
		limit = *req.DailyRequestLimit
	}
	tokenLimit := 0
	if req.DailyTokenLimit != nil {
		tokenLimit = *req.DailyTokenLimit
	}
	sortSeq := 0
	if req.SortSeq != nil {
		sortSeq = *req.SortSeq
	}
	row := &store.AIUpstreamEndpoint{
		EndpointID:          id,
		ProviderDisplayName: name,
		UpstreamBaseURL:     base,
		APISecretCipher:     cipher,
		ServiceEnabled:      enabled,
		DefaultModelCode:    strings.TrimSpace(req.DefaultModel),
		DailyRequestLimit:   limit,
		DailyTokenLimit:     tokenLimit,
		SortSeq:             sortSeq,
		CreatedByStaffID:    st.ID,
		UpdatedByStaffID:    st.ID,
	}
	if err := s.store.InsertAIEndpoint(r.Context(), row); err != nil {
		s.log.Error("create ai endpoint", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.endpoint.create", ResourceKind: "ai_endpoint", ResourceID: id,
		Detail: map[string]any{"name": name},
	})
	created, err := s.store.GetAIEndpoint(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	created.APISecretCipher = ""
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleOpsAIEndpointGet(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	e, err := s.store.GetAIEndpoint(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	e.APISecretCipher = ""
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleOpsAIEndpointSecret(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	e, err := s.store.GetAIEndpoint(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"found": strings.TrimSpace(e.APISecretCipher) != ""})
}

func (s *Server) handleOpsAIEndpointPatch(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	e, err := s.store.GetAIEndpoint(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	var req opsAIEndpointPatchReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.ProviderName != nil {
		name := strings.TrimSpace(*req.ProviderName)
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_endpoint")
			return
		}
		e.ProviderDisplayName = name
	}
	if req.BaseURL != nil {
		base := strings.TrimSpace(*req.BaseURL)
		if base == "" {
			writeError(w, http.StatusBadRequest, "invalid_endpoint")
			return
		}
		if err := llm.ValidateUpstreamURL(base); err != nil {
			writeError(w, http.StatusBadRequest, "unsafe_upstream")
			return
		}
		e.UpstreamBaseURL = base
	}
	if req.APISecret != nil {
		if secret := strings.TrimSpace(*req.APISecret); secret != "" {
			c, sealErr := s.sealSecret(secret)
			if sealErr != nil {
				writeError(w, http.StatusInternalServerError, "server_error")
				return
			}
			e.APISecretCipher = c
		}
	}
	if req.Enabled != nil {
		e.ServiceEnabled = *req.Enabled
	}
	if req.DefaultModel != nil {
		e.DefaultModelCode = strings.TrimSpace(*req.DefaultModel)
	}
	if req.DailyRequestLimit != nil {
		e.DailyRequestLimit = *req.DailyRequestLimit
	}
	if req.DailyTokenLimit != nil {
		e.DailyTokenLimit = *req.DailyTokenLimit
	}
	if req.SortSeq != nil {
		e.SortSeq = *req.SortSeq
	}
	e.UpdatedByStaffID = st.ID
	if err := s.store.UpdateAIEndpoint(r.Context(), e); err != nil {
		s.log.Error("patch ai endpoint", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.endpoint.patch", ResourceKind: "ai_endpoint", ResourceID: e.EndpointID,
		Detail: map[string]any{},
	})
	updated, err := s.store.GetAIEndpoint(r.Context(), e.EndpointID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	updated.APISecretCipher = ""
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleOpsAIEndpointDelete(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	id := r.PathValue("id")
	if err := s.store.DeleteAIEndpoint(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.endpoint.delete", ResourceKind: "ai_endpoint", ResourceID: id,
		Detail: map[string]any{},
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleOpsAIModelList(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	if _, err := s.store.GetAIEndpoint(r.Context(), r.PathValue("id")); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	items, err := s.store.ListAIModels(r.Context(), r.PathValue("id"), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.AIModelCatalog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleOpsAIModelCreate(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	ep, err := s.store.GetAIEndpoint(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	var req opsAIModelWriteReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeError(w, http.StatusBadRequest, "invalid_model")
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = code
	}
	status := strings.TrimSpace(req.RecordStatus)
	if status == "" {
		status = "active"
	}
	sortSeq := 0
	if req.SortSeq != nil {
		sortSeq = *req.SortSeq
	}
	id, err := auth.NewID("aimdl")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	row := &store.AIModelCatalog{
		CatalogID:        id,
		EndpointID:       ep.EndpointID,
		ModelCode:        code,
		ModelLabel:       label,
		RecordStatus:     status,
		SortSeq:          sortSeq,
		CreatedByStaffID: st.ID,
		UpdatedByStaffID: st.ID,
	}
	if err := s.store.InsertAIModel(r.Context(), row); errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "model_exists")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.model.create", ResourceKind: "ai_model", ResourceID: id,
		Detail: map[string]any{"code": code, "endpointId": ep.EndpointID},
	})
	created, err := s.store.GetAIModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleOpsAIModelPatch(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	m, err := s.store.GetAIModel(r.Context(), r.PathValue("modelId"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	var req opsAIModelWriteReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if code := strings.TrimSpace(req.Code); code != "" {
		m.ModelCode = code
	}
	if req.Label != "" || req.Code != "" {
		label := strings.TrimSpace(req.Label)
		if label == "" {
			label = m.ModelCode
		}
		m.ModelLabel = label
	}
	if status := strings.TrimSpace(req.RecordStatus); status != "" {
		m.RecordStatus = status
	}
	if req.SortSeq != nil {
		m.SortSeq = *req.SortSeq
	}
	m.UpdatedByStaffID = st.ID
	if err := s.store.UpdateAIModel(r.Context(), m); errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "model_exists")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.model.patch", ResourceKind: "ai_model", ResourceID: m.CatalogID,
		Detail: map[string]any{},
	})
	updated, err := s.store.GetAIModel(r.Context(), m.CatalogID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleOpsAIModelDelete(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	id := r.PathValue("modelId")
	if err := s.store.DeleteAIModel(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ai.model.delete", ResourceKind: "ai_model", ResourceID: id,
		Detail: map[string]any{},
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// resolvedAIUpstream 是 Ops 探测 / 拉模型时解析出的上游地址与密钥。
type resolvedAIUpstream struct {
	Base  string
	Key   string
	Model string
}

// resolveOpsAIUpstream 从表单或已存接入点取出可出站的 Base / 密钥 / 默认模型。
func (s *Server) resolveOpsAIUpstream(r *http.Request, endpointID, base, secret, model string) (resolvedAIUpstream, string) {
	out := resolvedAIUpstream{
		Base:  strings.TrimSpace(base),
		Key:   strings.TrimSpace(secret),
		Model: strings.TrimSpace(model),
	}
	if id := strings.TrimSpace(endpointID); id != "" {
		e, err := s.store.GetAIEndpoint(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			return out, "not_found"
		}
		if err != nil {
			s.log.Error("ai resolve endpoint", zap.Error(err))
			return out, "server_error"
		}
		if out.Base == "" {
			out.Base = e.UpstreamBaseURL
		}
		if out.Model == "" {
			out.Model = e.DefaultModelCode
		}
		if out.Key == "" {
			plain, openErr := s.openSecret(e.APISecretCipher)
			if openErr != nil {
				s.log.Error("ai resolve decrypt", zap.Error(openErr))
				return out, "server_error"
			}
			out.Key = strings.TrimSpace(plain)
		}
	}
	if out.Base == "" {
		return out, "invalid_endpoint"
	}
	if err := llm.ValidateUpstreamURL(out.Base); err != nil {
		return out, "unsafe_upstream"
	}
	if out.Key == "" {
		return out, "missing_secret"
	}
	return out, ""
}

func writeOpsAIResolveError(w http.ResponseWriter, code string) {
	switch code {
	case "not_found":
		writeError(w, http.StatusNotFound, code)
	case "server_error":
		writeError(w, http.StatusInternalServerError, code)
	case "invalid_endpoint", "unsafe_upstream", "missing_secret", "model_required":
		writeError(w, http.StatusBadRequest, code)
	default:
		writeError(w, http.StatusBadRequest, code)
	}
}

// handleOpsAIEndpointTest 用表单或已存密钥向上游发一条极短对话，供 Ops 验证连通。
func (s *Server) handleOpsAIEndpointTest(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	if s.aiTestCD != nil && !s.aiTestCD.Allow("ai-test|"+st.ID) {
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	var req opsAIEndpointTestReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	up, code := s.resolveOpsAIUpstream(r, req.EndpointID, req.BaseURL, req.APISecret, req.Model)
	if code != "" {
		writeOpsAIResolveError(w, code)
		return
	}
	if up.Model == "" {
		writeError(w, http.StatusBadRequest, "model_required")
		return
	}
	result := llm.ProbeChat(r.Context(), llm.Upstream{BaseURL: up.Base, APIKey: up.Key}, up.Model, llm.NewSafeClient(llm.ProbeTimeout))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        result.OK,
		"status":    result.Status,
		"latencyMs": result.Latency.Milliseconds(),
		"message":   result.Message,
		"model":     result.Model,
	})
}

// handleOpsAIEndpointPullModels 向 OpenAI 兼容上游 GET /models，供 Ops 选择目录。
func (s *Server) handleOpsAIEndpointPullModels(w http.ResponseWriter, r *http.Request) {
	st := s.requireStaff(w, r)
	if st == nil {
		return
	}
	if s.aiTestCD != nil && !s.aiTestCD.Allow("ai-test|"+st.ID) {
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	var req opsAIEndpointTestReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	up, code := s.resolveOpsAIUpstream(r, req.EndpointID, req.BaseURL, req.APISecret, req.Model)
	if code != "" {
		writeOpsAIResolveError(w, code)
		return
	}
	result := llm.ListModels(r.Context(), llm.Upstream{BaseURL: up.Base, APIKey: up.Key}, llm.NewSafeClient(llm.ProbeTimeout))
	items := result.Models
	if items == nil {
		items = []llm.RemoteModel{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        result.OK,
		"status":    result.Status,
		"latencyMs": result.Latency.Milliseconds(),
		"message":   result.Message,
		"items":     items,
	})
}

// handleOpsAIUsage 返回某 UTC 日系统模型用量（员工）。day 空则今天。
func (s *Server) handleOpsAIUsage(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	day, err := store.ParseUsageDay(r.URL.Query().Get("day"), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_day")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListAIUsage(r.Context(), store.AIUsageListFilter{
		Day:    day,
		UserID: r.URL.Query().Get("userId"),
		Email:  r.URL.Query().Get("email"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		s.log.Error("list ai usage", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.AIUsageRow{}
	}
	sum, err := s.store.SummarizeAIUsage(r.Context(), day)
	if err != nil {
		s.log.Error("summarize ai usage", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	reqLimit, tokenLimit := 0, 0
	if ep, epErr := s.store.GetActiveAIEndpoint(r.Context()); epErr == nil {
		reqLimit = ep.DailyRequestLimit
		tokenLimit = ep.DailyTokenLimit
	}
	trend, err := s.store.ListAIUsageTrend(r.Context(), day, 7)
	if err != nil {
		s.log.Error("ai usage trend", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	top, _, err := s.store.ListAIUsage(r.Context(), store.AIUsageListFilter{
		Day:    day,
		UserID: r.URL.Query().Get("userId"),
		Email:  r.URL.Query().Get("email"),
		Limit:  8,
		Offset: 0,
	})
	if err != nil {
		s.log.Error("ai usage top", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if top == nil {
		top = []store.AIUsageRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"day":               day,
		"summary":           sum,
		"dailyRequestLimit": reqLimit,
		"dailyTokenLimit":   tokenLimit,
		"trend":             trend,
		"top":               top,
		"items":             items,
		"total":             total,
	})
}
