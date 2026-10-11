package httpapi

import (
	"context"
	"errors"

	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

// latestDesktopRelease 在同一条桌面更新线上取 semver 最大的已发布版本。
// 版本相同时保留 ensis-ops 的记录。
func (s *Server) latestDesktopRelease(ctx context.Context, product, channel, platform, arch string) (*store.AppRelease, error) {
	var best *store.AppRelease
	for _, id := range desktopProductIDs(product) {
		rel, err := s.store.LatestPublishedRelease(ctx, id, channel, platform, arch)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if best == nil || newerDesktopRelease(rel, best) {
			best = rel
		}
	}
	if best == nil {
		return nil, store.ErrNotFound
	}
	return best, nil
}

// publishedDesktopRelease 按版本取已发布记录。同一版本两边都有时保留 ensis-ops。
func (s *Server) publishedDesktopRelease(ctx context.Context, product, channel, platform, arch, version string) (*store.AppRelease, error) {
	var chosen *store.AppRelease
	for _, id := range desktopProductIDs(product) {
		rel, err := s.store.GetPublishedReleaseByDim(ctx, id, channel, platform, arch, version)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if chosen == nil || rel.Product == "ensis-ops" {
			chosen = rel
		}
	}
	if chosen == nil {
		return nil, store.ErrNotFound
	}
	return chosen, nil
}

// listDesktopReleases 合并同一条桌面更新线的已发布版本，再按请求分页。
func (s *Server) listDesktopReleases(ctx context.Context, product, channel, platform, arch string, limit, offset int) ([]store.AppRelease, int, error) {
	ids := desktopProductIDs(product)
	if len(ids) == 1 {
		return s.store.ListPublishedReleases(ctx, product, channel, platform, arch, limit, offset)
	}
	merged := make(map[string]store.AppRelease)
	for _, id := range ids {
		items, _, err := s.store.ListPublishedReleases(ctx, id, channel, platform, arch, 50, 0)
		if err != nil {
			return nil, 0, err
		}
		for i := range items {
			item := items[i]
			if prev, ok := merged[item.Version]; !ok || (item.Product == "ensis-ops" && prev.Product != "ensis-ops") {
				merged[item.Version] = item
			}
		}
	}
	items := make([]store.AppRelease, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sortDesktopReleases(items)
	total := len(items)
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []store.AppRelease{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func newerDesktopRelease(rel, best *store.AppRelease) bool {
	cmp, err := store.CompareSemVerStrings(best.Version, rel.Version)
	if err != nil {
		return false
	}
	if cmp < 0 {
		return true
	}
	return cmp == 0 && rel.Product == "ensis-ops" && best.Product != "ensis-ops"
}

func (s *Server) sumDesktopHits(ctx context.Context, product string) (store.UpdateHitStats, error) {
	out := store.UpdateHitStats{ByPlatform: map[string]int64{}}
	for _, id := range desktopProductIDs(product) {
		st, err := s.store.SumUpdateHits(ctx, id)
		if err != nil {
			return out, err
		}
		out.Total += st.Total
		for platform, n := range st.ByPlatform {
			out.ByPlatform[platform] += n
		}
	}
	return out, nil
}

func sortDesktopReleases(items []store.AppRelease) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			cmp, err := store.CompareSemVerStrings(items[i].Version, items[j].Version)
			if err != nil {
				continue
			}
			if cmp < 0 {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
