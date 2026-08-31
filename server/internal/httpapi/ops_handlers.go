package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

const opsRefreshCookie = "niuma_ops_rt"

type opsLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleOpsLogin(w http.ResponseWriter, r *http.Request) {
	var req opsLoginReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	ip := s.requestIP(r)
	if s.authCD != nil && !s.authCD.Allow("ops-login|"+email+"|"+ip) {
		s.log.Warn("ops login rate limited", zap.String("email", email), zap.String("ip", ip))
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	st, err := s.store.GetStaffByEmail(r.Context(), email)
	if err != nil || st == nil || !auth.CheckPassword(st.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}
	if st.Status != "active" {
		writeError(w, http.StatusForbidden, "account_disabled")
		return
	}

	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: st.ID,
		ActionCode: "ops.login", ResourceKind: "staff", ResourceID: st.ID,
		Detail: map[string]any{"email": st.Email},
	})
	s.writeOpsAuth(w, r, st)
}

type opsRefreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

func (s *Server) handleOpsRefresh(w http.ResponseWriter, r *http.Request) {
	raw := opsRefreshFromRequest(r)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "invalid_refresh")
		return
	}
	hash := auth.HashToken(raw)
	sess, err := s.store.GetStaffSessionByHash(r.Context(), hash)
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
	st, err := s.store.GetStaffByID(r.Context(), sess.StaffID)
	if err != nil || st.Status != "active" {
		writeError(w, http.StatusUnauthorized, "invalid_refresh")
		return
	}
	_ = s.store.RevokeStaffSession(r.Context(), hash)
	s.writeOpsAuth(w, r, st)
}

func (s *Server) handleOpsLogout(w http.ResponseWriter, r *http.Request) {
	if raw := opsRefreshFromRequest(r); raw != "" {
		_ = s.store.RevokeStaffSession(r.Context(), auth.HashToken(raw))
	}
	s.clearOpsRefreshCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func opsRefreshFromRequest(r *http.Request) string {
	if c, err := r.Cookie(opsRefreshCookie); err == nil {
		if v := strings.TrimSpace(c.Value); v != "" {
			return v
		}
	}
	if r.Body == nil {
		return ""
	}
	var req opsRefreshReq
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		return ""
	}
	return strings.TrimSpace(req.RefreshToken)
}

func (s *Server) writeOpsAuth(w http.ResponseWriter, r *http.Request, st *store.StaffUser) {
	access, accessExp, err := s.tokens.Issue(st.ID, st.Email, auth.KindStaff)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	refresh, err := auth.RandomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	sid, err := auth.NewID("sfs")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC()
	if err := s.store.CreateStaffSession(r.Context(), sid, st.ID, auth.HashToken(refresh),
		now.Add(s.cfg.RefreshTokenTTL).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		s.log.Error("create staff session", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.setOpsRefreshCookie(w, r, refresh)
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": access,
		"expiresAt":   accessExp.UTC().Format(time.RFC3339),
		"staff":       store.PublicStaff(st),
	})
}

func (s *Server) cookieSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) setOpsRefreshCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     opsRefreshCookie,
		Value:    token,
		Path:     CloudPrefix,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure(r),
		MaxAge:   int(s.cfg.RefreshTokenTTL.Seconds()),
	})
}

func (s *Server) clearOpsRefreshCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     opsRefreshCookie,
		Value:    "",
		Path:     CloudPrefix,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure(r),
		MaxAge:   -1,
	})
}

func (s *Server) requireStaff(w http.ResponseWriter, r *http.Request) *store.StaffUser {
	tok := bearerToken(r)
	if tok == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	claims, err := s.tokens.Parse(tok)
	if err != nil || claims.Kind != auth.KindStaff {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	st, err := s.store.GetStaffByID(r.Context(), claims.SubjectID)
	if err != nil || st.Status != "active" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	return st
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *store.StaffUser {
	st := s.requireStaff(w, r)
	if st == nil {
		return nil
	}
	if !store.IsStaffAdmin(st) {
		writeError(w, http.StatusForbidden, "forbidden")
		return nil
	}
	return st
}

func (s *Server) guardLastAdmin(w http.ResponseWriter, r *http.Request, target *store.StaffUser, nextStatus, nextRole string) bool {
	if target == nil || !store.IsStaffAdmin(target) || target.Status != "active" {
		return true
	}
	demote := nextRole != "" && nextRole != store.StaffRoleAdmin
	disable := nextStatus == "disabled"
	if !demote && !disable {
		return true
	}
	n, err := s.store.CountActiveAdmins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return false
	}
	if n <= 1 {
		writeError(w, http.StatusBadRequest, "last_admin")
		return false
	}
	return true
}

func (s *Server) handleOpsStats(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	users, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	feedback, err := s.store.CountFeedback(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	week, err := s.store.CountFeedbackSince(r.Context(), time.Now().Add(-7*24*time.Hour))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	hits, err := s.store.CountUpdateHits(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	aiToday, err := s.store.SummarizeAIUsage(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users":           users,
		"feedback":        feedback,
		"feedbackLast7d":  week,
		"downloadHits":    hits,
		"aiUsersToday":    aiToday.UserCount,
		"aiRequestsToday": aiToday.RequestCount,
	})
}

func (s *Server) handleOpsUsers(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListUsers(r.Context(), store.UserListFilter{
		ID:          r.URL.Query().Get("id"),
		Email:       r.URL.Query().Get("email"),
		DisplayName: r.URL.Query().Get("displayName"),
		Status:      r.URL.Query().Get("status"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		s.log.Error("list users", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.UserListItem{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		m := store.PublicUser(&it.User)
		m["feedbackCount"] = it.FeedbackCount
		m["sessionCount"] = it.SessionCount
		out = append(out, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "total": total})
}

func (s *Server) handleOpsUserDetail(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	id := r.PathValue("id")
	u, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	fb, _ := s.store.ListFeedbackByUser(r.Context(), u.ID, 50)
	sessions, _ := s.store.CountActiveSessions(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":          store.PublicUser(u),
		"sessionCount":  sessions,
		"feedback":      fb,
	})
}

func (s *Server) handleOpsFeedbackList(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListFeedback(r.Context(), store.FeedbackListFilter{
		ID:            r.URL.Query().Get("id"),
		Title:         r.URL.Query().Get("title"),
		Contact:       r.URL.Query().Get("contact"),
		Category:      r.URL.Query().Get("category"),
		Status:        r.URL.Query().Get("status"),
		Product:       r.URL.Query().Get("product"),
		UserID:        r.URL.Query().Get("userId"),
		ClientVersion: r.URL.Query().Get("clientVersion"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.Feedback{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

type opsFeedbackPatch struct {
	Status     *string `json:"status"`
	AdminNote  *string `json:"adminNote"`
	StaffReply *string `json:"staffReply"`
}

func (s *Server) handleOpsFeedbackPatch(w http.ResponseWriter, r *http.Request) {
	staff := s.requireStaff(w, r)
	if staff == nil {
		return
	}
	var req opsFeedbackPatch
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	existing, err := s.store.GetFeedback(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}

	upd := store.FeedbackUpdate{
		Status:         existing.Status,
		AdminNote:      existing.AdminNote,
		StaffReply:     existing.StaffReply,
		StaffReplierID: existing.StaffReplierID,
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "" && status != "open" && status != "triaged" && status != "closed" {
			writeError(w, http.StatusBadRequest, "invalid_status")
			return
		}
		if status != "" {
			upd.Status = status
		}
	}
	if req.AdminNote != nil {
		upd.AdminNote = *req.AdminNote
	}
	if req.StaffReply != nil {
		upd.StaffReply = strings.TrimSpace(*req.StaffReply)
		upd.StaffReplierID = staff.ID
		upd.TouchReply = true
	}

	f, err := s.store.UpdateFeedback(r.Context(), existing.ID, upd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}

	action := "feedback.update"
	if upd.TouchReply && f.StaffReply != "" && f.StaffReply != existing.StaffReply {
		action = "feedback.reply"
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: staff.ID,
		ActionCode: action, ResourceKind: "feedback", ResourceID: f.ID,
		Detail: map[string]any{
			"status":       f.Status,
			"hasReply":     f.StaffReply != "",
			"replyChanged": upd.TouchReply,
		},
	})

	// 回复触达：
	// - 已登录用户（桌面）：仅站内 /feedback/mine，不发邮件
	// - 官网匿名：邮件通知 + 官网 lookup 查询
	if upd.TouchReply && f.StaffReply != "" && f.StaffReply != existing.StaffReply {
		if strings.TrimSpace(f.UserID) == "" {
			go s.notifyAnonymousFeedbackReply(f)
		}
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) notifyAnonymousFeedbackReply(f *store.Feedback) {
	if f == nil || strings.TrimSpace(f.Contact) == "" {
		return
	}
	if err := s.mailer.SendFeedbackReply(f.Contact, f.Title, f.StaffReply); err != nil {
		s.log.Warn("feedback reply mail failed", zap.Error(err), zap.String("id", f.ID))
		return
	}
	if err := s.store.MarkFeedbackReplyNotified(context.Background(), f.ID); err != nil {
		s.log.Warn("mark reply notified failed", zap.Error(err), zap.String("id", f.ID))
	}
}
