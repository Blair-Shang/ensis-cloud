package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/llm"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

type aiCatalogModel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type aiRuntime struct {
	Endpoint store.AIUpstreamEndpoint
	Models   []store.AIModelCatalog
	APIKey   string
}

func (s *Server) loadAIRuntime(r *http.Request) (*aiRuntime, error) {
	ep, err := s.store.GetActiveAIEndpoint(r.Context())
	if err != nil {
		return nil, err
	}
	key, err := s.openSecret(ep.APISecretCipher)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(ep.UpstreamBaseURL) == "" || strings.TrimSpace(key) == "" {
		return nil, store.ErrNotFound
	}
	if err := llm.ValidateUpstreamURL(ep.UpstreamBaseURL); err != nil {
		return nil, store.ErrNotFound
	}
	models, err := s.store.ListAIModels(r.Context(), ep.EndpointID, "active")
	if err != nil {
		return nil, err
	}
	return &aiRuntime{Endpoint: *ep, Models: models, APIKey: key}, nil
}

func (rt *aiRuntime) modelCodes() []string {
	if rt == nil {
		return nil
	}
	out := make([]string, 0, len(rt.Models))
	for _, m := range rt.Models {
		if code := strings.TrimSpace(m.ModelCode); code != "" {
			out = append(out, code)
		}
	}
	return out
}

// handleAICatalog 返回系统模型目录与当日配额，供桌面端写入 niuma-system。
func (s *Server) handleAICatalog(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	rt, err := s.loadAIRuntime(r)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	if err != nil {
		s.log.Error("ai runtime", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	used, err := s.store.GetAIUsage(r.Context(), u.ID, time.Now().UTC())
	if err != nil {
		s.log.Error("ai usage", zap.Error(err), zap.String("user", u.ID))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	models := make([]aiCatalogModel, 0, len(rt.Models))
	for _, m := range rt.Models {
		label := strings.TrimSpace(m.ModelLabel)
		if label == "" {
			label = m.ModelCode
		}
		models = append(models, aiCatalogModel{Code: m.ModelCode, Label: label})
	}
	if len(models) == 0 && rt.Endpoint.DefaultModelCode != "" {
		models = append(models, aiCatalogModel{Code: rt.Endpoint.DefaultModelCode, Label: rt.Endpoint.DefaultModelCode})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":           true,
		"providerName":      rt.Endpoint.ProviderDisplayName,
		"defaultModel":      rt.Endpoint.DefaultModelCode,
		"models":            models,
		"dailyRequestLimit": rt.Endpoint.DailyRequestLimit,
		"dailyTokenLimit":   rt.Endpoint.DailyTokenLimit,
		"requestsUsed":      used.RequestCount,
		"tokensUsed":        used.TokenCount,
	})
}

// handleAIChatCompletions 以用户登录态代理库中已开放接入点的上游对话。
func (s *Server) handleAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	rt, err := s.loadAIRuntime(r)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusServiceUnavailable, "ai_disabled")
		return
	}
	if err != nil {
		s.log.Error("ai runtime", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}

	if !s.aiChat.Acquire(u.ID) {
		writeError(w, http.StatusTooManyRequests, "ai_busy")
		return
	}
	defer s.aiChat.Release(u.ID)

	raw, err := io.ReadAll(io.LimitReader(r.Body, llm.MaxChatBodyBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	if len(raw) > llm.MaxChatBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "body_too_large")
		return
	}

	model := llm.PickModel(llm.RequestedModel(raw), rt.modelCodes(), rt.Endpoint.DefaultModelCode)
	if model == "" {
		writeError(w, http.StatusBadRequest, "model_required")
		return
	}
	body, err := llm.RewriteChatBody(raw, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	used, err := s.store.TryAddAIUsage(r.Context(), u.ID, 1, 0, rt.Endpoint.DailyRequestLimit, rt.Endpoint.DailyTokenLimit, time.Now().UTC())
	if errors.Is(err, store.ErrQuotaExceeded) {
		writeError(w, http.StatusTooManyRequests, "ai_quota_exceeded")
		return
	}
	if err != nil {
		s.log.Error("ai usage increment", zap.Error(err), zap.String("user", u.ID))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}

	s.log.Info("ai chat",
		zap.String("user", u.ID),
		zap.String("model", model),
		zap.Int("used", used.RequestCount),
	)

	if err := llm.ProxyChat(r.Context(), w, llm.Upstream{
		BaseURL: rt.Endpoint.UpstreamBaseURL,
		APIKey:  rt.APIKey,
	}, body, llm.NewSafeClient(llm.DefaultChatTimeout)); err != nil {
		s.log.Warn("ai proxy", zap.Error(err), zap.String("user", u.ID))
		if w.Header().Get("Content-Type") == "" {
			writeError(w, http.StatusBadGateway, "ai_upstream_error")
		}
	}
}
