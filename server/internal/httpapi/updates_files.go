package httpapi

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/Blair-Shang/ensis-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

// handleUpdatesFile 公开下载本机安装包；仅 published 记录对应文件。
// 大文件由 Nginx 限连接与带宽，本进程不按 IP 记账。
func (s *Server) handleUpdatesFile(w http.ResponseWriter, r *http.Request) {
	if s.files == nil || s.store == nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	dim := releasefiles.Dim{
		Product:  r.PathValue("product"),
		Channel:  r.PathValue("channel"),
		Platform: r.PathValue("platform"),
		Arch:     r.PathValue("arch"),
		Version:  strings.TrimPrefix(strings.TrimSpace(r.PathValue("version")), "v"),
		Name:     r.PathValue("name"),
	}
	dim, err := releasefiles.Normalize(dim)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if _, _, _, _, nerr := normalizeReleaseDims(dim.Product, dim.Channel, dim.Platform, dim.Arch); nerr != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	sv, err := store.ParseSemVer(dim.Version)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	dim.Version = sv.String()

	rel, err := s.store.GetReleaseByDim(r.Context(), dim.Product, dim.Channel, dim.Platform, dim.Arch, dim.Version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if rel.Status != "published" {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if !s.files.Exists(dim) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}

	f, err := os.Open(s.files.FilePath(dim))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+dim.Name+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, dim.Name, st.ModTime(), f)
}
