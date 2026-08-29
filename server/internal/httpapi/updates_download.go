package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

// handleUpdatesDownload 稳定的「最新包」入口：按查询维度解析当前 latest published，
// 记一次 update_hits，再 302 到带版本号的安装包地址。
//
// 官网 / 投放链接应指向本接口，而不是页面加载时缓存的 downloadUrl。
// 本响应禁止缓存，避免浏览器把「最新」钉在旧版本。不走 updateCD：
// 真实下载不应被清单探针冷却打断（与 handleUpdatesFile 一致）。
func (s *Server) handleUpdatesDownload(w http.ResponseWriter, r *http.Request) {
	product, channel, platform, arch, err := parseUpdateQuery(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.store == nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	rel, err := s.store.LatestPublishedRelease(r.Context(), product, channel, platform, arch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		s.log.Error("download latest release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	loc := s.safeDownloadRedirect(s.latestDownloadLocation(rel))
	if loc == "" {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if _, hitErr := s.store.IncrementUpdateHit(r.Context(), rel.Product, rel.Channel, rel.Platform, rel.Arch, rel.Version); hitErr != nil {
		s.log.Warn("download increment hit", zap.Error(hitErr),
			zap.String("version", rel.Version), zap.String("platform", rel.Platform))
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	http.Redirect(w, r, loc, http.StatusFound)
}

// latestDownloadLocation 优先本机已落盘安装包（官方 files 地址），否则用发布记录里的 downloadUrl。
func (s *Server) latestDownloadLocation(rel *store.AppRelease) string {
	if rel == nil {
		return ""
	}
	if loc := s.managedInstallerLocation(rel); loc != "" {
		return loc
	}
	return strings.TrimSpace(rel.DownloadURL)
}

// managedInstallerLocation 若本机有对应安装包，返回可公开访问的 files URL。
func (s *Server) managedInstallerLocation(rel *store.AppRelease) string {
	if s.files == nil || rel == nil {
		return ""
	}
	dim, ok := s.files.Resolve(releasefiles.Dim{
		Product:  rel.Product,
		Channel:  rel.Channel,
		Platform: rel.Platform,
		Arch:     rel.Arch,
		Version:  rel.Version,
		Name:     installerNameFromURL(rel.DownloadURL),
	})
	if !ok {
		return ""
	}
	base := strings.TrimSpace(s.cfg.PublicBase)
	if base == "" {
		base = CloudPrefix
	}
	return releasefiles.PublicURL(base, dim)
}

// safeDownloadRedirect 只允许跳到已校验的 https 包地址或本站 files 路径，避免开放重定向。
func (s *Server) safeDownloadRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isManagedInstallerURL(raw) {
		if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") && !strings.Contains(raw, "\\") {
			return raw
		}
		u, err := url.Parse(raw)
		if err == nil && u.Scheme == "https" && u.User == nil && u.Host != "" {
			return raw
		}
	}
	if err := validateDownloadURL(raw, s.cfg.DownloadURLHosts); err != nil {
		return ""
	}
	return raw
}

func installerNameFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	name, err := url.PathUnescape(path.Base(u.Path))
	if err != nil {
		return ""
	}
	if name == "." || name == "/" {
		return ""
	}
	return name
}
