-- 004_update_hits：桌面 / 官网更新包下载计次（SQLite）

CREATE TABLE IF NOT EXISTS update_download_hits (
	product_code TEXT NOT NULL,   -- 产品标识
	channel_code TEXT NOT NULL,   -- 渠道：stable/beta
	platform_code TEXT NOT NULL,  -- windows/linux/macos
	arch_code TEXT NOT NULL,      -- x64/arm64
	version_label TEXT NOT NULL,  -- semver
	hit_count INTEGER NOT NULL DEFAULT 0, -- 累计点击/拉起下载次数
	last_hit_at TEXT NOT NULL,    -- 最近一次计次时间
	PRIMARY KEY (product_code, channel_code, platform_code, arch_code, version_label)
);
