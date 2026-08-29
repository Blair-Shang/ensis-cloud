package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/feedbackmedia"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

type feedbackReq struct {
	Category      string                     `json:"category"`
	Title         string                     `json:"title"`
	Body          string                     `json:"body"`
	Contact       string                     `json:"contact"`
	Product       string                     `json:"product"`
	ClientVersion string                     `json:"clientVersion"`
	Images        []feedbackmedia.ImageInput `json:"images"`
}

func (s *Server) handleFeedbackCreate(w http.ResponseWriter, r *http.Request) {
	if s.fbCreateCD != nil && !s.fbCreateCD.Allow("fb-create:"+hashIP(s.requestIP(r))) {
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	var req feedbackReq
	if err := decodeJSONLimit(w, r, &req, 8<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	req.Category = strings.TrimSpace(req.Category)
	req.Contact = strings.TrimSpace(req.Contact)
	req.Product = strings.TrimSpace(req.Product)
	req.ClientVersion = strings.TrimSpace(req.ClientVersion)

	if req.Title == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "title_and_body_required")
		return
	}
	if utf8.RuneCountInString(req.Title) > 200 || utf8.RuneCountInString(req.Body) > 8000 {
		writeError(w, http.StatusBadRequest, "payload_too_large")
		return
	}

	user := s.optionalUser(r)
	userID := ""
	if user != nil {
		userID = user.ID
		if req.Contact == "" {
			req.Contact = user.Email
		}
	} else if req.Contact == "" {
		writeError(w, http.StatusBadRequest, "contact_required")
		return
	}

	id, err := auth.NewID("fb")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}

	atts, err := feedbackmedia.SaveImages(s.cfg.DataDir, id, req.Images)
	if err != nil {
		switch err.Error() {
		case "too_many_images", "image_too_large", "unsupported_image_type", "invalid_image":
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.log.Error("save feedback images", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "save_failed")
		}
		return
	}

	ipHash := hashIP(s.requestIP(r))
	f, err := s.store.InsertFeedback(r.Context(), store.FeedbackInput{
		ID:            id,
		UserID:        userID,
		Category:      req.Category,
		Title:         req.Title,
		Body:          req.Body,
		Contact:       req.Contact,
		Product:       req.Product,
		ClientVersion: req.ClientVersion,
		IPHash:        ipHash,
		Attachments:   atts,
	})
	if err != nil {
		feedbackmedia.RemoveDir(s.cfg.DataDir, id)
		s.log.Error("feedback insert", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "save_failed")
		return
	}
	actorKind, actorID := "guest", ""
	if userID != "" {
		actorKind, actorID = "user", userID
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: actorKind, ActorID: actorID,
		ActionCode: "feedback.create", ResourceKind: "feedback", ResourceID: f.ID,
		Detail: map[string]any{
			"category":        f.Category,
			"title":           f.Title,
			"attachmentCount": len(atts),
		},
		SourceIPHash: ipHash,
	})
	s.log.Info("feedback received",
		zap.String("id", f.ID),
		zap.String("category", f.Category),
		zap.Int("attachments", len(atts)),
	)
	if s.cfg.FeedbackWebhookURL != "" {
		go s.postFeedbackWebhook(f)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": f.ID, "ok": true, "status": f.Status})
}

func (s *Server) handleFeedbackMine(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	items, err := s.store.ListFeedbackByUser(r.Context(), u.ID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, store.PublicFeedback(it))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// handleFeedbackLookup 官网匿名查询回复：?id=&contact=
func (s *Server) handleFeedbackLookup(w http.ResponseWriter, r *http.Request) {
	if s.fbLookupCD != nil && !s.fbLookupCD.Allow("fb-lookup:"+hashIP(s.requestIP(r))) {
		writeError(w, http.StatusTooManyRequests, "rate_limited")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	contact := strings.TrimSpace(r.URL.Query().Get("contact"))
	if id == "" || contact == "" {
		writeError(w, http.StatusBadRequest, "id_and_contact_required")
		return
	}
	f, err := s.store.GetFeedbackByIDAndContact(r.Context(), id, contact)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	writeJSON(w, http.StatusOK, store.PublicFeedback(*f))
}

// handleFeedbackAttachment 下载反馈截图（工单所有者或员工）。
func (s *Server) handleFeedbackAttachment(w http.ResponseWriter, r *http.Request) {
	fbID := strings.TrimSpace(r.PathValue("id"))
	attID := strings.TrimSpace(r.PathValue("attId"))
	if fbID == "" || attID == "" {
		writeError(w, http.StatusBadRequest, "invalid_input")
		return
	}
	f, err := s.store.GetFeedback(r.Context(), fbID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}

	staff := s.optionalStaff(r)
	user := s.optionalUser(r)
	allowed := staff != nil || (user != nil && user.ID == f.UserID)
	if !allowed {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	att := feedbackmedia.FindAttachment(f.Attachments, attID)
	if att == nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	path := feedbackmedia.FilePath(s.cfg.DataDir, fbID, att.File)
	raw, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) optionalStaff(r *http.Request) *store.StaffUser {
	tok := bearerToken(r)
	if tok == "" {
		return nil
	}
	claims, err := s.tokens.Parse(tok)
	if err != nil || claims.Kind != auth.KindStaff {
		return nil
	}
	st, err := s.store.GetStaffByID(r.Context(), claims.SubjectID)
	if err != nil || st.Status != "active" {
		return nil
	}
	return st
}

func (s *Server) postFeedbackWebhook(f *store.Feedback) {
	payload, _ := json.Marshal(map[string]any{
		"id":            f.ID,
		"userId":        f.UserID,
		"category":      f.Category,
		"title":         f.Title,
		"body":          f.Body,
		"contact":       f.Contact,
		"product":       f.Product,
		"clientVersion": f.ClientVersion,
		"status":        f.Status,
		"attachments":   f.Attachments,
	})
	resp, err := s.client.Post(s.cfg.FeedbackWebhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		s.log.Warn("feedback webhook failed", zap.Error(err), zap.String("id", f.ID))
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		s.log.Warn("feedback webhook bad status", zap.Int("status", resp.StatusCode), zap.String("id", f.ID))
	}
}
