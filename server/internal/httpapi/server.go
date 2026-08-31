package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/config"
	"github.com/Blair-Shang/niuma-cloud/server/internal/mail"
	"github.com/Blair-Shang/niuma-cloud/server/internal/ratelimit"
	"github.com/Blair-Shang/niuma-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

// Server 聚合 Identity / Feedback / Ops 的 HTTP 入口。
type Server struct {
	cfg        config.Config
	store      *store.Store
	mailer     *mail.Mailer
	tokens     *auth.TokenIssuer
	otpCD      *ratelimit.CoolDown // OTP 发送冷却
	authCD     *ratelimit.CoolDown // 密码 / Ops 登录节流
	pipelineCD *ratelimit.CoolDown // 流水线推送节流
	fbCreateCD *ratelimit.CoolDown // 公开提单节流
	fbLookupCD *ratelimit.CoolDown // 匿名查询节流
	aiTestCD   *ratelimit.CoolDown // Ops 探测上游冷却
	aiChat     *ratelimit.Inflight // 同一用户并发对话路数
	files      *releasefiles.Store
	log        *zap.Logger
	client     *http.Client // 反馈 Webhook 等出站请求
	static     http.Handler // 嵌入的 Admin 前端（base=/niuma/ops/）
}

func New(cfg config.Config, st *store.Store, mailer *mail.Mailer, logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		cfg:        cfg,
		store:      st,
		mailer:     mailer,
		tokens:     auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTokenTTL),
		otpCD:      ratelimit.NewCoolDown(cfg.OTPCooldown),
		authCD:     ratelimit.NewCoolDown(2 * time.Second),
		pipelineCD: ratelimit.NewCoolDown(2 * time.Second),
		fbCreateCD: ratelimit.NewCoolDown(30 * time.Second),
		fbLookupCD: ratelimit.NewCoolDown(2 * time.Second),
		aiTestCD:   ratelimit.NewCoolDown(3 * time.Second),
		aiChat:     ratelimit.NewInflight(2),
		files:      releasefiles.New(cfg.ReleasesDir),
		log:        logger,
		client:     &http.Client{Timeout: 8 * time.Second},
	}
}

// SetStatic 注入 Admin 静态资源（构建产物或开发代理产物）。
func (s *Server) SetStatic(h http.Handler) {
	s.static = h
}

// SeedStaff 首次启动时按配置创建员工账号（已存在则跳过），并写审计。
func (s *Server) SeedStaff(ctx context.Context) error {
	if s.cfg.StaffEmail == "" || s.cfg.StaffPassword == "" {
		return nil
	}
	_, err := s.store.GetStaffByEmail(ctx, s.cfg.StaffEmail)
	if err == nil {
		return nil
	}
	if err != store.ErrNotFound {
		return err
	}
	hash, err := auth.HashPassword(s.cfg.StaffPassword)
	if err != nil {
		return err
	}
	id, err := auth.NewID("staff")
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.store.CreateStaff(ctx, store.StaffUser{
		ID:           id,
		Email:        s.cfg.StaffEmail,
		PasswordHash: hash,
		DisplayName:  "Admin",
		Status:       "active",
		Role:         store.StaffRoleAdmin,
		CreatedAt:    now,
	}); err != nil {
		return err
	}
	s.writeAudit(ctx, nil, store.AuditLog{
		ActorKind:    "system",
		ActorID:      "",
		ActionCode:   "staff.seed",
		ResourceKind: "staff",
		ResourceID:   id,
		Detail:       map[string]any{"email": s.cfg.StaffEmail},
	})
	return nil
}

// Handler 注册全部路由并包装 CORS。
//
// 规范路径（工程根 /niuma）：
//
//	GET  /niuma/cloud/healthz
//	POST /niuma/cloud/api/v1/auth/*          注册 / 登录 / OTP / 改密 / 刷新 / 登出
//	GET|PATCH /niuma/cloud/api/v1/me
//	POST /niuma/cloud/api/v1/feedback
//	GET  /niuma/cloud/api/v1/feedback/mine|lookup
//	GET  /niuma/cloud/api/v1/updates/latest|check|releases
//	GET  /niuma/cloud/api/v1/updates/download   最新 published，302 到版本化安装包
//	POST /niuma/cloud/api/v1/updates/hit
//	GET  /niuma/cloud/api/v1/updates/files/...  本机安装包（仅 published）
//	POST /niuma/cloud/api/v1/pipeline/releases/assets  流水线上传安装包
//	POST /niuma/cloud/api/v1/pipeline/releases  流水线登记并发布
//	GET  /niuma/cloud/api/v1/ai/catalog
//	POST /niuma/cloud/api/v1/ai/v1/chat/completions
//	POST /niuma/cloud/api/v1/ops/ai/endpoints/test  Ops 探测上游
//	POST /niuma/cloud/api/v1/ops/ai/endpoints/pull-models  Ops 拉上游模型目录
//	*    /niuma/cloud/api/v1/ops/ai/*            系统模型接入点 / 目录 CRUD
//	POST /niuma/cloud/api/v1/ops/auth/login|refresh|logout|password
//	GET  /niuma/cloud/api/v1/ops/stats|audit-logs|staff|users|feedback|releases
//	GET  /niuma/cloud/api/v1/ops/ai/usage     系统模型按日用量
//	*    /niuma/ops/*                        Admin SPA
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerCloudAPI(mux, CloudPrefix)

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		path := r.URL.Path
		if isCloudAPIPath(path) {
			mux.ServeHTTP(w, r)
			return
		}
		if s.serveOps(w, r) {
			return
		}
		if isServiceHintPath(path) {
			writeJSON(w, http.StatusOK, map[string]any{
				"service": "niuma-cloud",
				"hint":    "use path prefixes",
				"root":    RootPrefix,
				"cloud":   CloudPrefix,
				"ops":     OpsPrefix,
			})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	})
	return s.withAccessLog(s.withCORS(root))
}

// registerCloudAPI 把 API 挂到 /niuma/cloud。
func (s *Server) registerCloudAPI(mux *http.ServeMux, p string) {
	mux.HandleFunc("GET "+p+"/healthz", s.handleHealth)

	mux.HandleFunc("GET "+p+"/api/v1/updates/latest", s.handleUpdatesLatest)
	mux.HandleFunc("GET "+p+"/api/v1/updates/check", s.handleUpdatesCheck)
	mux.HandleFunc("GET "+p+"/api/v1/updates/releases", s.handleUpdatesReleases)
	mux.HandleFunc("GET "+p+"/api/v1/updates/download", s.handleUpdatesDownload)
	mux.HandleFunc("POST "+p+"/api/v1/updates/hit", s.handleUpdatesHit)
	mux.HandleFunc("GET "+p+"/api/v1/updates/hits", s.handleUpdatesHitStats)
	mux.HandleFunc("GET "+p+"/api/v1/updates/files/{product}/{channel}/{platform}/{arch}/{version}/{name}", s.handleUpdatesFile)
	mux.HandleFunc("POST "+p+"/api/v1/pipeline/releases/assets", s.handlePipelineReleaseAsset)
	mux.HandleFunc("POST "+p+"/api/v1/pipeline/releases", s.handlePipelineReleases)

	mux.HandleFunc("POST "+p+"/api/v1/auth/register/start", s.handleRegisterStart)
	mux.HandleFunc("POST "+p+"/api/v1/auth/register/complete", s.handleRegisterComplete)
	mux.HandleFunc("POST "+p+"/api/v1/auth/login", s.handleLoginPassword)
	mux.HandleFunc("POST "+p+"/api/v1/auth/login/otp/start", s.handleLoginOTPStart)
	mux.HandleFunc("POST "+p+"/api/v1/auth/login/otp/complete", s.handleLoginOTPComplete)
	mux.HandleFunc("POST "+p+"/api/v1/auth/password/forgot", s.handlePasswordForgot)
	mux.HandleFunc("POST "+p+"/api/v1/auth/password/reset", s.handlePasswordReset)
	mux.HandleFunc("POST "+p+"/api/v1/auth/password/change", s.handlePasswordChange)
	mux.HandleFunc("POST "+p+"/api/v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST "+p+"/api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET "+p+"/api/v1/me", s.handleMe)
	mux.HandleFunc("PATCH "+p+"/api/v1/me", s.handlePatchMe)

	mux.HandleFunc("GET "+p+"/api/v1/ai/catalog", s.handleAICatalog)
	mux.HandleFunc("POST "+p+"/api/v1/ai/v1/chat/completions", s.handleAIChatCompletions)

	mux.HandleFunc("POST "+p+"/api/v1/feedback", s.handleFeedbackCreate)
	mux.HandleFunc("GET "+p+"/api/v1/feedback/mine", s.handleFeedbackMine)
	mux.HandleFunc("GET "+p+"/api/v1/feedback/lookup", s.handleFeedbackLookup)
	mux.HandleFunc("GET "+p+"/api/v1/feedback/{id}/attachments/{attId}", s.handleFeedbackAttachment)

	mux.HandleFunc("POST "+p+"/api/v1/ops/auth/login", s.handleOpsLogin)
	mux.HandleFunc("POST "+p+"/api/v1/ops/auth/refresh", s.handleOpsRefresh)
	mux.HandleFunc("POST "+p+"/api/v1/ops/auth/logout", s.handleOpsLogout)
	mux.HandleFunc("POST "+p+"/api/v1/ops/auth/password", s.handleOpsChangePassword)
	mux.HandleFunc("GET "+p+"/api/v1/ops/stats", s.handleOpsStats)
	mux.HandleFunc("GET "+p+"/api/v1/ops/audit-logs", s.handleOpsAuditLogs)
	mux.HandleFunc("GET "+p+"/api/v1/ops/staff", s.handleOpsStaffList)
	mux.HandleFunc("POST "+p+"/api/v1/ops/staff", s.handleOpsStaffCreate)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/staff/{id}", s.handleOpsStaffPatch)
	mux.HandleFunc("POST "+p+"/api/v1/ops/staff/{id}/password", s.handleOpsStaffResetPassword)
	mux.HandleFunc("GET "+p+"/api/v1/ops/users", s.handleOpsUsers)
	mux.HandleFunc("POST "+p+"/api/v1/ops/users", s.handleOpsUserCreate)
	mux.HandleFunc("GET "+p+"/api/v1/ops/users/{id}", s.handleOpsUserDetail)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/users/{id}", s.handleOpsUserPatch)
	mux.HandleFunc("POST "+p+"/api/v1/ops/users/{id}/password", s.handleOpsUserResetPassword)
	mux.HandleFunc("GET "+p+"/api/v1/ops/feedback", s.handleOpsFeedbackList)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/feedback/{id}", s.handleOpsFeedbackPatch)
	mux.HandleFunc("GET "+p+"/api/v1/ops/releases", s.handleOpsReleaseList)
	mux.HandleFunc("POST "+p+"/api/v1/ops/releases", s.handleOpsReleaseCreate)
	mux.HandleFunc("GET "+p+"/api/v1/ops/releases/{id}", s.handleOpsReleaseGet)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/releases/{id}", s.handleOpsReleasePatch)
	mux.HandleFunc("POST "+p+"/api/v1/ops/releases/{id}/publish", s.handleOpsReleasePublish)
	mux.HandleFunc("POST "+p+"/api/v1/ops/releases/{id}/yank", s.handleOpsReleaseYank)

	mux.HandleFunc("GET "+p+"/api/v1/ops/ai/usage", s.handleOpsAIUsage)
	mux.HandleFunc("GET "+p+"/api/v1/ops/ai/endpoints", s.handleOpsAIEndpointList)
	mux.HandleFunc("POST "+p+"/api/v1/ops/ai/endpoints", s.handleOpsAIEndpointCreate)
	mux.HandleFunc("POST "+p+"/api/v1/ops/ai/endpoints/test", s.handleOpsAIEndpointTest)
	mux.HandleFunc("POST "+p+"/api/v1/ops/ai/endpoints/pull-models", s.handleOpsAIEndpointPullModels)
	mux.HandleFunc("GET "+p+"/api/v1/ops/ai/endpoints/{id}", s.handleOpsAIEndpointGet)
	mux.HandleFunc("GET "+p+"/api/v1/ops/ai/endpoints/{id}/secret", s.handleOpsAIEndpointSecret)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/ai/endpoints/{id}", s.handleOpsAIEndpointPatch)
	mux.HandleFunc("DELETE "+p+"/api/v1/ops/ai/endpoints/{id}", s.handleOpsAIEndpointDelete)
	mux.HandleFunc("GET "+p+"/api/v1/ops/ai/endpoints/{id}/models", s.handleOpsAIModelList)
	mux.HandleFunc("POST "+p+"/api/v1/ops/ai/endpoints/{id}/models", s.handleOpsAIModelCreate)
	mux.HandleFunc("PATCH "+p+"/api/v1/ops/ai/models/{modelId}", s.handleOpsAIModelPatch)
	mux.HandleFunc("DELETE "+p+"/api/v1/ops/ai/models/{modelId}", s.handleOpsAIModelDelete)
}

// serveOps 提供 Admin SPA（/niuma/ops）。
func (s *Server) serveOps(w http.ResponseWriter, r *http.Request) bool {
	if !isOpsPath(r.URL.Path) {
		return false
	}
	if s.static == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return true
	}
	r2 := r.Clone(r.Context())
	trimmed := strings.TrimPrefix(r.URL.Path, OpsPrefix)
	if trimmed == "" {
		trimmed = "/"
	}
	r2.URL.Path = trimmed
	s.static.ServeHTTP(w, r2)
	return true
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "niuma-cloud"})
}

// withCORS 按配置放行 Origin；处理浏览器预检 OPTIONS。
func (s *Server) withCORS(next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, o := range s.cfg.CORSOrigins {
		allowed[o] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-Label")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
