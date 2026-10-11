package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/ensis-cloud/server/internal/auth"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

func (s *Server) handleOpsAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListAuditLogs(r.Context(), store.AuditListFilter{
		ActionCode:   r.URL.Query().Get("action"),
		ActorKind:    r.URL.Query().Get("actorKind"),
		ActorID:      r.URL.Query().Get("actorId"),
		ResourceKind: r.URL.Query().Get("resourceKind"),
		ResourceID:   r.URL.Query().Get("resourceId"),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		s.log.Error("list audit logs", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.AuditLogItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (s *Server) handleOpsStaffList(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListStaff(r.Context(), store.StaffListFilter{
		ID:          r.URL.Query().Get("id"),
		Email:       r.URL.Query().Get("email"),
		DisplayName: r.URL.Query().Get("displayName"),
		Status:      r.URL.Query().Get("status"),
		Role:        r.URL.Query().Get("role"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		s.log.Error("list staff", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, store.PublicStaff(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "total": total})
}

type opsStaffCreateReq struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

func (s *Server) handleOpsStaffCreate(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	var req opsStaffCreateReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	if err := store.EnsureEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_email")
		return
	}
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	id, err := auth.NewID("staff")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	display := strings.TrimSpace(req.DisplayName)
	if display == "" {
		display = strings.Split(email, "@")[0]
	}
	st := store.StaffUser{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  display,
		Status:       "active",
		Role:         store.NormalizeStaffRole(req.Role),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateStaff(r.Context(), st); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "email_taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "staff.create", ResourceKind: "staff", ResourceID: id,
		Detail: map[string]any{"email": email, "displayName": display, "role": st.Role},
	})
	writeJSON(w, http.StatusCreated, store.PublicStaff(&st))
}

type opsStaffPatchReq struct {
	DisplayName *string `json:"displayName"`
	Status      *string `json:"status"`
	Role        *string `json:"role"`
}

func (s *Server) handleOpsStaffPatch(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	id := r.PathValue("id")
	target, err := s.store.GetStaffByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	var req opsStaffPatchReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	display := target.DisplayName
	status := target.Status
	role := store.NormalizeStaffRole(target.Role)
	if req.DisplayName != nil {
		display = strings.TrimSpace(*req.DisplayName)
	}
	if req.Status != nil {
		st := strings.TrimSpace(*req.Status)
		if st != "active" && st != "disabled" {
			writeError(w, http.StatusBadRequest, "invalid_status")
			return
		}
		if st == "disabled" && id == actor.ID {
			writeError(w, http.StatusBadRequest, "cannot_disable_self")
			return
		}
		status = st
	}
	if req.Role != nil {
		role = store.NormalizeStaffRole(*req.Role)
	}
	if !s.guardLastAdmin(w, r, target, status, role) {
		return
	}
	if err := s.store.UpdateStaffProfile(r.Context(), id, display, status, role); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	updated, _ := s.store.GetStaffByID(r.Context(), id)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "staff.update", ResourceKind: "staff", ResourceID: id,
		Detail: map[string]any{"displayName": display, "status": status, "role": role},
	})
	writeJSON(w, http.StatusOK, store.PublicStaff(updated))
}

type opsPasswordSetReq struct {
	Password string `json:"password"`
}

func (s *Server) handleOpsStaffResetPassword(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	id := r.PathValue("id")
	if _, err := s.store.GetStaffByID(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	var req opsPasswordSetReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if err := s.store.UpdateStaffPassword(r.Context(), id, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	_ = s.store.RevokeStaffSessionsByStaffID(r.Context(), id)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "staff.password_reset", ResourceKind: "staff", ResourceID: id,
		Detail: map[string]any{},
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type opsChangePasswordReq struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleOpsChangePassword(w http.ResponseWriter, r *http.Request) {
	actor := s.requireStaff(w, r)
	if actor == nil {
		return
	}
	var req opsChangePasswordReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !auth.CheckPassword(actor.PasswordHash, req.CurrentPassword) {
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
	if err := s.store.UpdateStaffPassword(r.Context(), actor.ID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	_ = s.store.RevokeStaffSessionsByStaffID(r.Context(), actor.ID)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "staff.password_change", ResourceKind: "staff", ResourceID: actor.ID,
		Detail: map[string]any{},
	})
	s.writeOpsAuth(w, r, actor)
}

type opsUserCreateReq struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (s *Server) handleOpsUserCreate(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	var req opsUserCreateReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	email := normalizeEmail(req.Email)
	if err := store.EnsureEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_email")
		return
	}
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password")
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
	display := strings.TrimSpace(req.DisplayName)
	if display == "" {
		display = strings.Split(email, "@")[0]
	}
	u := store.User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  display,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	verified := now
	u.EmailVerifiedAt = &verified
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, "email_taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "user.create", ResourceKind: "user", ResourceID: id,
		Detail: map[string]any{"email": email},
	})
	writeJSON(w, http.StatusCreated, store.PublicUser(&u))
}

type opsUserPatchReq struct {
	Status *string `json:"status"`
}

func (s *Server) handleOpsUserPatch(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	id := r.PathValue("id")
	u, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	var req opsUserPatchReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.Status == nil {
		writeJSON(w, http.StatusOK, store.PublicUser(u))
		return
	}
	status := strings.TrimSpace(*req.Status)
	if status != "active" && status != "disabled" {
		writeError(w, http.StatusBadRequest, "invalid_status")
		return
	}
	if err := s.store.UpdateUserStatus(r.Context(), id, status); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	u.Status = status
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "user.status_update", ResourceKind: "user", ResourceID: id,
		Detail: map[string]any{"status": status},
	})
	writeJSON(w, http.StatusOK, store.PublicUser(u))
}

func (s *Server) handleOpsUserResetPassword(w http.ResponseWriter, r *http.Request) {
	actor := s.requireAdmin(w, r)
	if actor == nil {
		return
	}
	id := r.PathValue("id")
	if _, err := s.store.GetUserByID(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	var req opsPasswordSetReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if !validPassword(req.Password) {
		writeError(w, http.StatusBadRequest, "invalid_password")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), id, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	_ = s.store.RevokeRefreshByUserID(r.Context(), id)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "user.password_reset", ResourceKind: "user", ResourceID: id,
		Detail: map[string]any{},
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
