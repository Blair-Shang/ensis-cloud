-- 004_update_hits：桌面 / 官网更新包下载计次（不经 cloud 反代字节流）

CREATE TABLE IF NOT EXISTS update_download_hits (
	product_code VARCHAR(64) NOT NULL COMMENT '产品标识',
	channel_code VARCHAR(32) NOT NULL COMMENT '渠道：stable/beta',
	platform_code VARCHAR(32) NOT NULL COMMENT 'windows/linux/macos',
	arch_code VARCHAR(32) NOT NULL COMMENT 'x64/arm64',
	version_label VARCHAR(64) NOT NULL COMMENT 'semver',
	hit_count BIGINT NOT NULL DEFAULT 0 COMMENT '累计点击/拉起下载次数',
	last_hit_at DATETIME(3) NOT NULL COMMENT '最近一次计次时间 UTC',
	PRIMARY KEY (product_code, channel_code, platform_code, arch_code, version_label)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='更新下载计次';
