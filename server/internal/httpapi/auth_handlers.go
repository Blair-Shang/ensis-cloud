package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

type emailReq struct {
	Email string `json:"email"`
}

func (s *Server) issueOTP(w http.ResponseWriter, r *http.Request, purpose string) {
	var req emailReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	if err := store.EnsureEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_email")
		return
	}

	key := purpose + "|" + email + "|" + s.requestIP(r)
	if !s.otpCD.Allow(key) {
		s.log.Warn("auth otp cooldown",
			zap.String("email", email),
			zap.String("purpose", purpose),
			zap.String("ip", s.requestIP(r)),
		)
		writeError(w, http.StatusTooManyRequests, "otp_cooldown")
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	n, err := s.store.CountOTPSince(r.Context(), email, purpose, since)
	if err != nil {
		s.log.Error("count otp", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if n >= s.cfg.OTPDailyLimit {
		s.log.Warn("auth otp daily limit",
			zap.String("email", email),
			zap.String("purpose", purpose),
			zap.Int("limit", s.cfg.OTPDailyLimit),
		)
		writeError(w, http.StatusTooManyRequests, "otp_daily_limit")
		return
	}

	switch purpose {
	case "register":
		if _, err := s.store.GetUserByEmail(r.Context(), email); err == nil {
			// 防枚举：已注册也返回 ok，不发信
			s.log.Info("auth otp skipped",
				zap.String("email", email),
				zap.String("purpose", purpose),
				zap.String("reason", "email_taken"),
			)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "expiresInSec": int(s.cfg.OTPTTL.Seconds())})
			return
		} else if err != store.ErrNotFound {
			s.log.Error("get user", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "server_error")
			return
		}
	case "reset":
		if _, err := s.store.GetUserByEmail(r.Context(), email); err == store.ErrNotFound {
			s.log.Info("auth otp skipped",
				zap.String("email", email),
				zap.String("purpose", purpose),
				zap.String("reason", "account_not_found"),
			)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "expiresInSec": int(s.cfg.OTPTTL.Seconds())})
			return
		} else if err != nil {
			s.log.Error("get user", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "server_error")
			return
		}
	}
	// login：无论账号是否存在都发码；complete 时不存在则建号。

	code, err := auth.GenerateOTP()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	id, err := auth.NewID("otp")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC()
	otp := store.OTP{
		ID:        id,
		Email:     email,
		Purpose:   purpose,
		CodeHash:  auth.HashOTP(code),
		ExpiresAt: now.Add(s.cfg.OTPTTL).Format(time.RFC3339Nano),
		CreatedAt: now.Format(time.RFC3339Nano),
	}
	if err := s.store.InsertOTP(r.Context(), otp); err != nil {
		s.log.Error("insert otp", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if err := s.mailer.SendOTP(email, purpose, code); err != nil {
		s.log.Warn("send otp mail", zap.Error(err))
		writeError(w, http.StatusBadGateway, "mail_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "expiresInSec": int(s.cfg.OTPTTL.Seconds())})
}

func (s *Server) handleRegisterStart(w http.ResponseWriter, r *http.Request) {
	s.issueOTP(w, r, "register")
}

func (s *Server) handleLoginOTPStart(w http.ResponseWriter, r *http.Request) {
	s.issueOTP(w, r, "login")
}

func (s *Server) handlePasswordForgot(w http.ResponseWriter, r *http.Request) {
	s.issueOTP(w, r, "reset")
}

type registerCompleteReq struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (s *Server) handleRegisterComplete(w http.ResponseWriter, r *http.Request) {
	var req registerCompleteReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	if err := store.EnsureEmail(email); err != nil || !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_input")
		return
	}
	if err := s.store.ConsumeOTP(r.Context(), email, "register", auth.HashOTP(strings.TrimSpace(req.Code))); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_otp")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	id, err := auth.NewID("usr")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		name = displayNameFromEmail(email)
	} else if utf8.RuneCountInString(name) > 64 {
		name = string([]rune(name)[:64])
	}
	u := store.User{
		ID:              id,
		Email:           email,
		PasswordHash:    hash,
		DisplayName:     name,
		EmailVerifiedAt: &now,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "email_taken")
			return
		}
		s.log.Error("create user", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "user", ActorID: u.ID,
		ActionCode: "auth.register", ResourceKind: "user", ResourceID: u.ID,
		Detail: map[string]any{"email": u.Email},
	})
	s.respondAuth(w, r, &u)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLoginPassword(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	ip := s.requestIP(r)
	if !s.authCD.Allow("login|" + email + "|" + ip) {
		s.log.Warn("auth login rate limited", zap.String("email", email), zap.String("ip", ip))
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	u, err := s.store.GetUserByEmail(r.Context(), email)
	if err != nil || u == nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		s.log.Warn("auth login failed",
			zap.String("method", "password"),
			zap.String("email", email),
			zap.String("ip", ip),
			zap.String("reason", "invalid_credentials"),
		)
		writeError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}
	if u.Status != "active" {
		s.log.Warn("auth login failed",
			zap.String("method", "password"),
			zap.String("email", email),
			zap.String("ip", ip),
			zap.String("reason", "account_disabled"),
		)
		writeError(w, http.StatusForbidden, "account_disabled")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "user", ActorID: u.ID,
		ActionCode: "auth.login.password", ResourceKind: "user", ResourceID: u.ID,
		Detail: map[string]any{"email": u.Email},
	})
	s.log.Info("auth login ok",
		zap.String("method", "password"),
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
		zap.String("ip", ip),
	)
	s.respondAuth(w, r, u)
}

type otpCompleteReq struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (s *Server) handleLoginOTPComplete(w http.ResponseWriter, r *http.Request) {
	var req otpCompleteReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	ip := s.requestIP(r)
	if err := s.store.ConsumeOTP(r.Context(), email, "login", auth.HashOTP(strings.TrimSpace(req.Code))); err != nil {
		s.log.Warn("auth login failed",
			zap.String("method", "otp"),
			zap.String("email", email),
			zap.String("ip", ip),
			zap.String("reason", "invalid_otp"),
		)
		writeError(w, http.StatusBadRequest, "invalid_otp")
		return
	}
	u, created, err := s.provisionUserForOTPLogin(r.Context(), email)
	if err != nil {
		s.log.Error("auth otp provision", zap.Error(err), zap.String("email", email))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if u.Status != "active" {
		s.log.Warn("auth login failed",
			zap.String("method", "otp"),
			zap.String("email", email),
			zap.String("ip", ip),
			zap.String("reason", "account_disabled"),
		)
		writeError(w, http.StatusForbidden, "account_disabled")
		return
	}
	if created {
		s.writeAudit(r.Context(), r, store.AuditLog{
			ActorKind: "user", ActorID: u.ID,
			ActionCode: "auth.register", ResourceKind: "user", ResourceID: u.ID,
			Detail: map[string]any{"email": u.Email, "via": "otp_login"},
		})
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "user", ActorID: u.ID,
		ActionCode: "auth.login.otp", ResourceKind: "user", ResourceID: u.ID,
		Detail: map[string]any{"email": u.Email, "created": created},
	})
	s.log.Info("auth login ok",
		zap.String("method", "otp"),
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
		zap.Bool("created", created),
		zap.String("ip", ip),
	)
	s.respondAuth(w, r, u)
}

// displayNameFromEmail 用邮箱本地部分作为默认显示名，最长 64 字符。
func displayNameFromEmail(email string) string {
	name := strings.SplitN(email, "@", 2)[0]
	name = strings.TrimSpace(name)
	if name == "" {
		name = "user"
	}
	if utf8.RuneCountInString(name) > 64 {
		return string([]rune(name)[:64])
	}
	return name
}

// provisionUserForOTPLogin 验证码校验通过后查找用户；不存在则创建无可用登录密码的账户。
// 并发建号冲突时回读已有用户。新用户之后可通过「忘记密码」设置密码。
func (s *Server) provisionUserForOTPLogin(ctx context.Context, email string) (*store.User, bool, error) {
	u, err := s.store.GetUserByEmail(ctx, email)
	if err == nil {
		return u, false, nil
	}
	if err != store.ErrNotFound {
		return nil, false, err
	}
	secret, err := auth.RandomToken(32)
	if err != nil {
		return nil, false, err
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return nil, false, err
	}
	id, err := auth.NewID("usr")
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	created := store.User{
		ID:              id,
		Email:           email,
		PasswordHash:    hash,
		DisplayName:     displayNameFromEmail(email),
		EmailVerifiedAt: &now,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.CreateUser(ctx, created); err != nil {
		if err == store.ErrConflict {
			existing, getErr := s.store.GetUserByEmail(ctx, email)
			return existing, false, getErr
		}
		return nil, false, err
	}
	return &created, true, nil
}

type resetReq struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (s *Server) handlePasswordReset(w http.ResponseWriter, r *http.Request) {
	var req resetReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password")
		return
	}
	if err := s.store.ConsumeOTP(r.Context(), email, "reset", auth.HashOTP(strings.TrimSpace(req.Code))); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_otp")
		return
	}
	u, err := s.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_otp")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	_ = s.store.RevokeRefreshByUserID(r.Context(), u.ID)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "user", ActorID: u.ID,
		ActionCode: "auth.password.reset", ResourceKind: "user", ResourceID: u.ID,
		Detail: map[string]any{"email": u.Email},
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type changePasswordReq struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// handlePasswordChange 登录用户修改自己的密码（校验当前密码）。
func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var req changePasswordReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusForbidden, "invalid_credentials")
		return
	}
	if !validPassword(req.NewPassword) {
		writeError(w, http.StatusBadRequest, "invalid_password")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	_ = s.store.RevokeRefreshByUserID(r.Context(), u.ID)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "user", ActorID: u.ID,
		ActionCode: "auth.password.change", ResourceKind: "user", ResourceID: u.ID,
		Detail: map[string]any{},
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	hash := auth.HashToken(strings.TrimSpace(req.RefreshToken))
	sess, err := s.store.GetRefreshByHash(r.Context(), hash)
	if err != nil || sess.RevokedAt != nil {
		writeError(w, http.StatusUnauthorized, "invalid_refresh")
		return
	}
	exp, err := time.Parse(time.RFC3339Nano, sess.ExpiresAt)
	if err != nil {
		exp, _ = time.Parse(time.RFC3339, sess.ExpiresAt)
	}
	if time.Now().After(exp) {
		writeError(w, http.StatusUnauthorized, "invalid_refresh")
		return
	}
	u, err := s.store.GetUserByID(r.Context(), sess.UserID)
	if err != nil || u.Status != "active" {
		writeError(w, http.StatusUnauthorized, "invalid_refresh")
		return
	}
	_ = s.store.RevokeRefresh(r.Context(), hash)
	access, accessExp, err := s.tokens.Issue(u.ID, u.Email, auth.KindUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	refresh, err := auth.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	sid, err := auth.NewID("rfs")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC()
	if err := s.store.CreateRefreshSession(r.Context(), store.RefreshSession{
		ID:          sid,
		UserID:      u.ID,
		TokenHash:   auth.HashToken(refresh),
		DeviceLabel: sess.DeviceLabel,
		ExpiresAt:   now.Add(s.cfg.RefreshTokenTTL).Format(time.RFC3339Nano),
		CreatedAt:   now.Format(time.RFC3339Nano),
	}); err != nil {
		s.log.Error("rotate refresh", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"expiresAt":    accessExp.UTC().Format(time.RFC3339),
		"user":         store.PublicUser(u),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	_ = s.store.RevokeRefresh(r.Context(), auth.HashToken(strings.TrimSpace(req.RefreshToken)))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	writeJSON(w, http.StatusOK, store.PublicUser(u))
}

type patchMeReq struct {
	DisplayName string `json:"displayName"`
}

func (s *Server) handlePatchMe(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var req patchMeReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		writeError(w, http.StatusBadRequest, "invalid_display_name")
		return
	}
	if err := s.store.UpdateUserDisplayName(r.Context(), u.ID, name); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	u.DisplayName = name
	writeJSON(w, http.StatusOK, store.PublicUser(u))
}

func (s *Server) respondAuth(w http.ResponseWriter, r *http.Request, u *store.User) {
	access, accessExp, err := s.tokens.Issue(u.ID, u.Email, auth.KindUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	refresh, err := auth.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	sid, err := auth.NewID("rfs")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC()
	device := strings.TrimSpace(r.Header.Get("X-Device-Label"))
	if utf8.RuneCountInString(device) > 64 {
		device = string([]rune(device)[:64])
	}
	sess := store.RefreshSession{
		ID:          sid,
		UserID:      u.ID,
		TokenHash:   auth.HashToken(refresh),
		DeviceLabel: device,
		ExpiresAt:   now.Add(s.cfg.RefreshTokenTTL).Format(time.RFC3339Nano),
		CreatedAt:   now.Format(time.RFC3339Nano),
	}
	if err := s.store.CreateRefreshSession(r.Context(), sess); err != nil {
		s.log.Error("create refresh", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"expiresAt":    accessExp.UTC().Format(time.RFC3339),
		"user":         store.PublicUser(u),
	})
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *store.User {
	tok := bearerToken(r)
	if tok == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	claims, err := s.tokens.Parse(tok)
	if err != nil || claims.Kind != auth.KindUser {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	u, err := s.store.GetUserByID(r.Context(), claims.SubjectID)
	if err != nil || u.Status != "active" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	return u
}

func (s *Server) optionalUser(r *http.Request) *store.User {
	tok := bearerToken(r)
	if tok == "" {
		return nil
	}
	claims, err := s.tokens.Parse(tok)
	if err != nil || claims.Kind != auth.KindUser {
		return nil
	}
	u, err := s.store.GetUserByID(r.Context(), claims.SubjectID)
	if err != nil || u.Status != "active" {
		return nil
	}
	return u
}
