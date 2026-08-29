-- 003_app_releases：桌面端 / 官网共用的应用发布记录（SQLite）

CREATE TABLE IF NOT EXISTS app_releases (
	id TEXT PRIMARY KEY,                              -- 发布记录 ID
	product_code TEXT NOT NULL DEFAULT 'niuma',       -- 产品标识
	version_label TEXT NOT NULL,                      -- semver，如 1.0.1
	channel_code TEXT NOT NULL DEFAULT 'stable',      -- 渠道：stable
	platform_code TEXT NOT NULL,                      -- windows/linux/macos
	arch_code TEXT NOT NULL,                          -- x64/arm64
	min_supported_version TEXT NOT NULL DEFAULT '',   -- 低于此版本强制更新
	force_update INTEGER NOT NULL DEFAULT 0,          -- 本条强制更新 0/1
	title_text TEXT NOT NULL DEFAULT '',              -- 短标题
	notes_md TEXT NOT NULL DEFAULT '',                -- Markdown 更新说明
	download_url TEXT NOT NULL,                       -- HTTPS 安装包直链
	sha256_hex TEXT NOT NULL,                         -- 小写 hex
	file_size INTEGER NOT NULL DEFAULT 0,             -- 字节
	release_status TEXT NOT NULL DEFAULT 'draft',     -- draft/published/yanked
	published_at TEXT,                                -- 发布时间
	created_by TEXT NOT NULL DEFAULT '',              -- 员工 ID
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE (product_code, channel_code, platform_code, arch_code, version_label)
);

CREATE INDEX IF NOT EXISTS idx_app_releases_latest
	ON app_releases(product_code, channel_code, platform_code, arch_code, release_status, published_at);
CREATE INDEX IF NOT EXISTS idx_app_releases_status ON app_releases(release_status);
