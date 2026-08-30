package store

import (
	"context"
	"strings"
)

// IncrementUpdateHit 对指定版本累加一次下载/拉起计数，返回最新累计。
func (s *Store) IncrementUpdateHit(ctx context.Context, product, channel, platform, arch, version string) (int64, error) {
	product = strings.TrimSpace(product)
	channel = strings.TrimSpace(channel)
	platform = strings.TrimSpace(platform)
	arch = strings.TrimSpace(arch)
	version = strings.TrimSpace(version)
	if product == "" || channel == "" || platform == "" || arch == "" || version == "" {
		return 0, ErrInvalidRelease
	}
	now := nowUTC()
	var q string
	if s.driver == "mysql" {
		q = `
			INSERT INTO update_download_hits (
				product_code, channel_code, platform_code, arch_code, version_label, hit_count, last_hit_at
			) VALUES (?, ?, ?, ?, ?, 1, ?)
			ON DUPLICATE KEY UPDATE hit_count = hit_count + 1, last_hit_at = VALUES(last_hit_at)`
	} else {
		q = `
			INSERT INTO update_download_hits (
				product_code, channel_code, platform_code, arch_code, version_label, hit_count, last_hit_at
			) VALUES (?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT(product_code, channel_code, platform_code, arch_code, version_label)
			DO UPDATE SET hit_count = hit_count + 1, last_hit_at = excluded.last_hit_at`
	}
	if _, err := s.db.ExecContext(ctx, q, product, channel, platform, arch, version, now); err != nil {
		return 0, err
	}
	var n int64
	err := s.db.QueryRowContext(ctx, `
		SELECT hit_count FROM update_download_hits
		WHERE product_code = ? AND channel_code = ? AND platform_code = ? AND arch_code = ? AND version_label = ?`,
		product, channel, platform, arch, version,
	).Scan(&n)
	return n, err
}

// CountUpdateHits 返回全部版本的下载计次总和。
func (s *Store) CountUpdateHits(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(hit_count), 0) FROM update_download_hits`).Scan(&n)
	return n, err
}

// UpdateHitStats 公开累计下载：官网 hit 与桌面自动更新共用同一本账。
type UpdateHitStats struct {
	Total      int64
	ByPlatform map[string]int64
}

// SumUpdateHits 按产品汇总 hit_count（含各平台）。product 空则按 niuma。
func (s *Store) SumUpdateHits(ctx context.Context, product string) (UpdateHitStats, error) {
	product = strings.TrimSpace(product)
	if product == "" {
		product = "niuma"
	}
	out := UpdateHitStats{ByPlatform: map[string]int64{}}
	rows, err := s.db.QueryContext(ctx, `
		SELECT platform_code, COALESCE(SUM(hit_count), 0)
		FROM update_download_hits
		WHERE product_code = ?
		GROUP BY platform_code`, product)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var platform string
		var n int64
		if err := rows.Scan(&platform, &n); err != nil {
			return out, err
		}
		out.ByPlatform[platform] = n
		out.Total += n
	}
	return out, rows.Err()
}
