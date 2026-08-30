package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

type pipelineReleaseItem struct {
	Platform    string `json:"platform"`
	Arch        string `json:"arch"`
	Name        string `json:"name"`
	DownloadURL string `json:"downloadUrl"`
	SHA256      string `json:"sha256"`
	FileSize    int64  `json:"fileSize"`
}

type pipelineReleasesReq struct {
	Product             string                `json:"product"`
	Version             string                `json:"version"`
	Channel             string                `json:"channel"`
	Title               string                `json:"title"`
	NotesMd             string                `json:"notesMd"`
	MinSupportedVersion string                `json:"minSupportedVersion"`
	ForceUpdate         bool                  `json:"forceUpdate"`
	Publish             *bool                 `json:"publish"`
	Items               []pipelineReleaseItem `json:"items"`
}

// requirePipelineToken 校验流水线共享密钥；未配置则关闭接口。
func (s *Server) requirePipelineToken(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(s.cfg.PipelineToken) == "" {
		writeError(w, http.StatusServiceUnavailable, "pipeline_disabled")
		return false
	}
	got := bearerToken(r)
	if got == "" {
		got = strings.TrimSpace(r.Header.Get("X-Niuma-Pipeline-Token"))
	}
	if !secureTokenEqual(s.cfg.PipelineToken, got) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

func secureTokenEqual(want, got string) bool {
	if want == "" || got == "" {
		return false
	}
	a := sha256.Sum256([]byte(want))
	b := sha256.Sum256([]byte(got))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

// handlePipelineReleases 流水线专用：按版本维度 upsert 并默认直接 publish。
// 已校验 pipeline token，不再按 IP 做 2s 冷却。一次发版会连登 beta 与 stable，冷却会误伤第二次。
func (s *Server) handlePipelineReleases(w http.ResponseWriter, r *http.Request) {
	if !s.requirePipelineToken(w, r) {
		return
	}
	var req pipelineReleasesReq
	if err := decodeJSONLimit(w, r, &req, 256<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items_required")
		return
	}
	if len(req.Items) > 16 {
		writeError(w, http.StatusBadRequest, "too_many_items")
		return
	}
	publish := true
	if req.Publish != nil {
		publish = *req.Publish
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		ver := strings.TrimSpace(req.Version)
		if ver != "" {
			title = "NiuMa " + strings.TrimPrefix(ver, "v")
		}
	}

	out := make([]store.AppRelease, 0, len(req.Items))
	for i := range req.Items {
		item := req.Items[i]
		rel, created, err := s.upsertPipelineRelease(r, req, item, title, publish)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrReleaseConflict):
				writeError(w, http.StatusConflict, "version_taken")
			case errors.Is(err, errReleaseYanked):
				writeError(w, http.StatusConflict, "release_yanked")
			default:
				if code := err.Error(); strings.HasPrefix(code, "invalid_") ||
					code == "download_url_https_required" ||
					code == "download_url_host_forbidden" ||
					code == "download_url_host_not_allowed" ||
					code == "download_url_credentials_forbidden" ||
					code == "notes_too_large" ||
					code == "missing_installer" {
					writeError(w, http.StatusBadRequest, code)
					return
				}
				s.log.Error("pipeline upsert release", zap.Error(err), zap.Int("item", i))
				writeError(w, http.StatusInternalServerError, "server_error")
			}
			return
		}
		action := "release.pipeline.update"
		if created {
			action = "release.pipeline.create"
		}
		s.writeAudit(r.Context(), r, store.AuditLog{
			ActorKind: "pipeline", ActorID: "github-actions",
			ActionCode: action, ResourceKind: "release", ResourceID: rel.ID,
			Detail: map[string]any{
				"version": rel.Version, "platform": rel.Platform, "arch": rel.Arch,
				"status": rel.Status, "downloadHost": downloadHostOnly(rel.DownloadURL),
			},
		})
		out = append(out, *rel)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "published": publish})
}

var errReleaseYanked = errors.New("release_yanked")

func (s *Server) upsertPipelineRelease(
	r *http.Request,
	req pipelineReleasesReq,
	item pipelineReleaseItem,
	title string,
	publish bool,
) (*store.AppRelease, bool, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rel := &store.AppRelease{
		Product:             req.Product,
		Version:             req.Version,
		Channel:             req.Channel,
		Platform:            item.Platform,
		Arch:                item.Arch,
		MinSupportedVersion: req.MinSupportedVersion,
		ForceUpdate:         req.ForceUpdate,
		Title:               title,
		NotesMd:             req.NotesMd,
		DownloadURL:         item.DownloadURL,
		SHA256:              item.SHA256,
		FileSize:            item.FileSize,
		Status:              "draft",
		CreatedBy:           "pipeline",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if publish {
		rel.Status = "published"
		rel.PublishedAt = &now
	}
	if err := validateReleaseMeta(rel); err != nil {
		return nil, false, err
	}
	if err := s.attachLocalInstaller(rel, item.Name); err != nil {
		return nil, false, err
	}
	if !isManagedInstallerURL(rel.DownloadURL) {
		if err := validateDownloadURL(rel.DownloadURL, s.cfg.DownloadURLHosts); err != nil {
			return nil, false, err
		}
	}
	rel.DownloadURL = strings.TrimSpace(rel.DownloadURL)
	existing, err := s.store.GetReleaseByDim(r.Context(), rel.Product, rel.Channel, rel.Platform, rel.Arch, rel.Version)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, false, err
	}
	if existing == nil {
		id, err := auth.NewID("rel")
		if err != nil {
			return nil, false, err
		}
		rel.ID = id
		if err := s.store.InsertRelease(r.Context(), rel); err != nil {
			return nil, false, err
		}
		return rel, true, nil
	}
	if existing.Status == "yanked" {
		return nil, false, errReleaseYanked
	}
	rel.ID = existing.ID
	rel.CreatedAt = existing.CreatedAt
	rel.CreatedBy = existing.CreatedBy
	if existing.Status == "published" && existing.PublishedAt != nil && publish {
		rel.PublishedAt = existing.PublishedAt
		rel.Status = "published"
	}
	if err := s.store.UpdateRelease(r.Context(), rel); err != nil {
		return nil, false, err
	}
	return rel, false, nil
}

// attachLocalInstaller 若本机已有安装包，则把 downloadUrl 写成官方站地址。
func (s *Server) attachLocalInstaller(rel *store.AppRelease, fileName string) error {
	if rel == nil {
		return fmt.Errorf("invalid_release")
	}
	if s.files != nil {
		dim, ok := s.files.Resolve(releasefiles.Dim{
			Product:  rel.Product,
			Channel:  rel.Channel,
			Platform: rel.Platform,
			Arch:     rel.Arch,
			Version:  rel.Version,
			Name:     fileName,
		})
		if ok {
			rel.DownloadURL = releasefiles.PublicURL(s.cfg.PublicBase, dim)
			if rel.FileSize <= 0 {
				if st, err := os.Stat(s.files.FilePath(dim)); err == nil {
					rel.FileSize = st.Size()
				}
			}
			return nil
		}
	}
	if strings.TrimSpace(rel.DownloadURL) == "" {
		return fmt.Errorf("missing_installer")
	}
	return nil
}

func isManagedInstallerURL(raw string) bool {
	return strings.Contains(strings.TrimSpace(raw), "/api/v1/updates/files/")
}
