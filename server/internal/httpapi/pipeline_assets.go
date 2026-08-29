package httpapi

import (
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

const maxPipelineMultipartMemory = 8 << 20 // 仅缓冲表单字段；文件走磁盘

// handlePipelineReleaseAsset 流水线上传一个安装包到本机 data/releases。
// 不做 pipelineCD：大文件上传耗时长，冷却会误伤重试。
func (s *Server) handlePipelineReleaseAsset(w http.ResponseWriter, r *http.Request) {
	if !s.requirePipelineToken(w, r) {
		return
	}
	if s.files == nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	max := s.cfg.MaxInstallerBytes
	if max <= 0 {
		max = 2 << 30
	}
	r.Body = http.MaxBytesReader(w, r.Body, max+maxPipelineMultipartMemory+1)
	if err := r.ParseMultipartForm(maxPipelineMultipartMemory); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_multipart")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file_required")
		return
	}
	defer file.Close()

	dim := releasefiles.Dim{
		Product:  firstForm(r, "product", "niuma"),
		Channel:  firstForm(r, "channel", "stable"),
		Platform: r.FormValue("platform"),
		Arch:     r.FormValue("arch"),
		Version:  strings.TrimPrefix(strings.TrimSpace(r.FormValue("version")), "v"),
		Name:     hdr.Filename,
	}
	if v := strings.TrimSpace(r.FormValue("name")); v != "" {
		dim.Name = v
	}
	dim, err = releasefiles.Normalize(dim)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, _, _, nerr := normalizeReleaseDims(dim.Product, dim.Channel, dim.Platform, dim.Arch); nerr != nil {
		writeError(w, http.StatusBadRequest, nerr.Error())
		return
	}

	size, sum, err := s.files.Save(dim, io.LimitReader(file, max+1), max, r.FormValue("sha256"))
	if err != nil {
		switch err.Error() {
		case "installer_too_large", "hash_mismatch", "invalid_installer_name", "invalid_version", "invalid_release_path":
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.log.Error("save installer", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "save_failed")
		}
		return
	}
	url := releasefiles.PublicURL(s.cfg.PublicBase, dim)
	s.writeAudit(r.Context(), r, store.AuditLog{
		ActorKind: "pipeline", ActorID: "github-actions",
		ActionCode: "release.pipeline.upload", ResourceKind: "release-file",
		ResourceID: dim.Product + "/" + dim.Version + "/" + dim.Platform + "/" + dim.Arch,
		Detail:     map[string]any{"name": dim.Name, "fileSize": size, "sha256": sum},
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"downloadUrl": url,
		"sha256":      sum,
		"fileSize":    size,
		"name":        dim.Name,
		"product":     dim.Product,
		"channel":     dim.Channel,
		"platform":    dim.Platform,
		"arch":        dim.Arch,
		"version":     dim.Version,
	})
}

func firstForm(r *http.Request, key, fallback string) string {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return fallback
	}
	return v
}
