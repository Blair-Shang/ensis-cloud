package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

type opsReleaseCreateReq struct {
	Product             string `json:"product"`
	Version             string `json:"version"`
	Channel             string `json:"channel"`
	Platform            string `json:"platform"`
	Arch                string `json:"arch"`
	MinSupportedVersion string `json:"minSupportedVersion"`
	ForceUpdate         bool   `json:"forceUpdate"`
	Title               string `json:"title"`
	NotesMd             string `json:"notesMd"`
	DownloadURL         string `json:"downloadUrl"`
	SHA256              string `json:"sha256"`
	FileSize            int64  `json:"fileSize"`
}

type opsReleasePatchReq struct {
	MinSupportedVersion *string `json:"minSupportedVersion"`
	ForceUpdate         *bool   `json:"forceUpdate"`
	Title               *string `json:"title"`
	NotesMd             *string `json:"notesMd"`
	DownloadURL         *string `json:"downloadUrl"`
	SHA256              *string `json:"sha256"`
	FileSize            *int64  `json:"fileSize"`
}

func (s *Server) handleOpsReleaseList(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := s.store.ListReleases(r.Context(), store.ReleaseListFilter{
		Product:  r.URL.Query().Get("product"),
		Channel:  r.URL.Query().Get("channel"),
		Platform: r.URL.Query().Get("platform"),
		Arch:     r.URL.Query().Get("arch"),
		Status:   r.URL.Query().Get("status"),
		Version:  r.URL.Query().Get("version"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		s.log.Error("list releases", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if items == nil {
		items = []store.AppRelease{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (s *Server) handleOpsReleaseGet(w http.ResponseWriter, r *http.Request) {
	if s.requireStaff(w, r) == nil {
		return
	}
	rel, err := s.store.GetReleaseByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (s *Server) handleOpsReleaseCreate(w http.ResponseWriter, r *http.Request) {
	actor := s.requireStaff(w, r)
	if actor == nil {
		return
	}
	var req opsReleaseCreateReq
	if err := decodeJSONLimit(w, r, &req, 256<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	id, err := auth.NewID("rel")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rel := &store.AppRelease{
		ID:                  id,
		Product:             req.Product,
		Version:             req.Version,
		Channel:             req.Channel,
		Platform:            req.Platform,
		Arch:                req.Arch,
		MinSupportedVersion: req.MinSupportedVersion,
		ForceUpdate:         req.ForceUpdate,
		Title:               req.Title,
		NotesMd:             req.NotesMd,
		DownloadURL:         req.DownloadURL,
		SHA256:              req.SHA256,
		FileSize:            req.FileSize,
		Status:              "draft",
		CreatedBy:           actor.ID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := validateReleaseFields(rel, s.cfg.DownloadURLHosts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.InsertRelease(r.Context(), rel); err != nil {
		if errors.Is(err, store.ErrReleaseConflict) {
			writeError(w, http.StatusConflict, "version_taken")
			return
		}
		s.log.Error("create release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "release.create", ResourceKind: "release", ResourceID: id,
		Detail: map[string]any{
			"version": rel.Version, "platform": rel.Platform, "arch": rel.Arch,
		},
	})
	writeJSON(w, http.StatusCreated, rel)
}

// handleOpsReleasePatch 改单条内核的安装包字段；标题与更新说明同步到同版本全部未撤回内核。
func (s *Server) handleOpsReleasePatch(w http.ResponseWriter, r *http.Request) {
	actor := s.requireStaff(w, r)
	if actor == nil {
		return
	}
	rel, err := s.store.GetReleaseByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if rel.Status == "yanked" {
		writeError(w, http.StatusConflict, "release_yanked")
		return
	}
	var req opsReleasePatchReq
	if err := decodeJSONLimit(w, r, &req, 256<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.MinSupportedVersion != nil {
		rel.MinSupportedVersion = *req.MinSupportedVersion
	}
	if req.ForceUpdate != nil {
		rel.ForceUpdate = *req.ForceUpdate
	}
	if req.Title != nil {
		rel.Title = *req.Title
	}
	if req.NotesMd != nil {
		rel.NotesMd = *req.NotesMd
	}
	if req.DownloadURL != nil {
		rel.DownloadURL = *req.DownloadURL
	}
	if req.SHA256 != nil {
		rel.SHA256 = *req.SHA256
	}
	if req.FileSize != nil {
		rel.FileSize = *req.FileSize
	}
	rel.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := validateReleaseFields(rel, s.cfg.DownloadURLHosts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.UpdateRelease(r.Context(), rel); err != nil {
		s.log.Error("patch release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	synced := 1
	if req.Title != nil || req.NotesMd != nil {
		notes := store.ReleaseNotesPatch{}
		if req.Title != nil {
			notes.Title = &rel.Title
		}
		if req.NotesMd != nil {
			notes.NotesMd = &rel.NotesMd
		}
		n, err := s.store.UpdateReleaseNotesByVersion(r.Context(), rel.Product, rel.Version, rel.UpdatedAt, notes)
		if err != nil {
			s.log.Error("patch release notes", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "server_error")
			return
		}
		synced = n
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "release.update", ResourceKind: "release", ResourceID: rel.ID,
		Detail: map[string]any{
			"version": rel.Version, "status": rel.Status, "notesSynced": synced,
		},
	})
	writeJSON(w, http.StatusOK, rel)
}

func (s *Server) handleOpsReleasePublish(w http.ResponseWriter, r *http.Request) {
	actor := s.requireStaff(w, r)
	if actor == nil {
		return
	}
	rel, err := s.store.GetReleaseByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if rel.Status == "published" {
		writeJSON(w, http.StatusOK, rel)
		return
	}
	if rel.Status == "yanked" {
		writeError(w, http.StatusConflict, "release_yanked")
		return
	}
	// 再验一次下载 URL，防止草稿期配置变更后放行危险地址
	if err := validateReleaseFields(rel, s.cfg.DownloadURLHosts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rel.Status = "published"
	rel.PublishedAt = &now
	rel.UpdatedAt = now
	if err := s.store.UpdateRelease(r.Context(), rel); err != nil {
		s.log.Error("publish release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "release.publish", ResourceKind: "release", ResourceID: rel.ID,
		Detail: map[string]any{
			"version": rel.Version, "platform": rel.Platform, "arch": rel.Arch,
			// 审计只记 host，不把完整 URL 当可执行入口回显到日志滥用面
			"downloadHost": downloadHostOnly(rel.DownloadURL),
		},
	})
	writeJSON(w, http.StatusOK, rel)
}

func (s *Server) handleOpsReleaseYank(w http.ResponseWriter, r *http.Request) {
	actor := s.requireStaff(w, r)
	if actor == nil {
		return
	}
	rel, err := s.store.GetReleaseByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if rel.Status == "yanked" {
		writeJSON(w, http.StatusOK, rel)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rel.Status = "yanked"
	rel.UpdatedAt = now
	if err := s.store.UpdateRelease(r.Context(), rel); err != nil {
		s.log.Error("yank release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "staff", ActorID: actor.ID,
		ActionCode: "release.yank", ResourceKind: "release", ResourceID: rel.ID,
		Detail: map[string]any{"version": rel.Version},
	})
	writeJSON(w, http.StatusOK, rel)
}

func downloadHostOnly(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexAny(raw, "/?"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}
