package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

// 公开 updates API：仅返回 published 元数据，永不返回 draft/createdBy。
// 服务端不拉取 download_url（无 SSRF）。

func (s *Server) handleUpdatesLatest(w http.ResponseWriter, r *http.Request) {
	product, channel, platform, arch, err := parseUpdateQuery(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rel, err := s.store.LatestPublishedRelease(r.Context(), product, channel, platform, arch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		s.log.Error("latest release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, store.PublicRelease(rel))
}

func (s *Server) handleUpdatesCheck(w http.ResponseWriter, r *http.Request) {
	product, channel, platform, arch, err := parseUpdateQuery(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	current := strings.TrimSpace(r.URL.Query().Get("current"))
	if _, err := store.ParseSemVer(current); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_current")
		return
	}
	rel, err := s.store.LatestPublishedRelease(r.Context(), product, channel, platform, arch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{
				"updateAvailable": false,
				"forceUpdate":     false,
				"current":         current,
				"latest":          nil,
			})
			return
		}
		s.log.Error("check release", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	cmp, err := store.CompareSemVerStrings(current, rel.Version)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_current")
		return
	}
	available := cmp < 0
	force := false
	if available {
		force = computeForceUpdate(current, rel)
	}
	var latest any
	if available {
		latest = store.PublicRelease(rel)
	} else {
		latest = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"updateAvailable": available,
		"forceUpdate":     force,
		"current":         current,
		"latest":          latest,
	})
}

func (s *Server) handleUpdatesReleases(w http.ResponseWriter, r *http.Request) {
	product, channel, platform, arch, err := parseUpdateQuery(r, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.store.ListPublishedReleases(r.Context(), product, channel, platform, arch, limit)
	if err != nil {
		s.log.Error("list published releases", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, store.PublicRelease(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func parseUpdateQuery(r *http.Request, platformOptional bool) (product, channel, platform, arch string, err error) {
	q := r.URL.Query()
	product = q.Get("product")
	channel = q.Get("channel")
	platform = q.Get("platform")
	arch = q.Get("arch")
	if !platformOptional && strings.TrimSpace(platform) == "" {
		return "", "", "", "", errors.New("invalid_platform")
	}
	if strings.TrimSpace(arch) == "" {
		return "", "", "", "", errors.New("invalid_arch")
	}
	return normalizeReleaseDims(product, channel, platform, arch)
}

type updateHitReq struct {
	Product  string `json:"product"`
	Channel  string `json:"channel"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
}

// handleUpdatesHit 记录一次安装包拉起/下载（不代理文件）。version 空则记最新 published。
func (s *Server) handleUpdatesHit(w http.ResponseWriter, r *http.Request) {
	var req updateHitReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	product, channel, platform, arch, err := normalizeReleaseDims(req.Product, req.Channel, req.Platform, req.Arch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		rel, lookErr := s.store.LatestPublishedRelease(r.Context(), product, channel, platform, arch)
		if lookErr != nil {
			if errors.Is(lookErr, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not_found")
				return
			}
			s.log.Error("hit latest release", zap.Error(lookErr))
			writeError(w, http.StatusInternalServerError, "server_error")
			return
		}
		version = rel.Version
	} else {
		sv, err := store.ParseSemVer(version)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_version")
			return
		}
		version = sv.String()
	}
	n, err := s.store.IncrementUpdateHit(r.Context(), product, channel, platform, arch, version)
	if err != nil {
		s.log.Error("increment update hit", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"hits":     n,
		"product":  product,
		"channel":  channel,
		"platform": platform,
		"arch":     arch,
		"version":  version,
	})
}

// handleUpdatesHitStats 公开累计下载次数：官网 /updates/download 与桌面 POST /updates/hit 同一本账。
func (s *Server) handleUpdatesHitStats(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	product := strings.TrimSpace(r.URL.Query().Get("product"))
	if product == "" {
		product = "niuma"
	}
	st, err := s.store.SumUpdateHits(r.Context(), product)
	if err != nil {
		s.log.Error("sum update hits", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=15")
	writeJSON(w, http.StatusOK, map[string]any{
		"product":    product,
		"total":      st.Total,
		"byPlatform": st.ByPlatform,
	})
}
