-- 003_app_releases：桌面端 / 官网共用的应用发布记录（MySQL）

CREATE TABLE IF NOT EXISTS app_releases (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '发布记录 ID',
	product_code VARCHAR(64) NOT NULL DEFAULT 'niuma' COMMENT '产品标识',
	version_label VARCHAR(64) NOT NULL COMMENT 'semver，如 1.0.1',
	channel_code VARCHAR(32) NOT NULL DEFAULT 'stable' COMMENT '渠道：stable',
	platform_code VARCHAR(32) NOT NULL COMMENT 'windows/linux/macos',
	arch_code VARCHAR(32) NOT NULL COMMENT 'x64/arm64',
	min_supported_version VARCHAR(64) NOT NULL DEFAULT '' COMMENT '低于此版本强制更新',
	force_update TINYINT(1) NOT NULL DEFAULT 0 COMMENT '本条强制更新',
	title_text VARCHAR(256) NOT NULL DEFAULT '' COMMENT '短标题',
	notes_md MEDIUMTEXT NOT NULL COMMENT 'Markdown 更新说明',
	download_url VARCHAR(2048) NOT NULL COMMENT 'HTTPS 安装包直链',
	sha256_hex CHAR(64) NOT NULL COMMENT '小写 hex',
	file_size BIGINT NOT NULL DEFAULT 0 COMMENT '字节',
	release_status VARCHAR(32) NOT NULL DEFAULT 'draft' COMMENT 'draft/published/yanked',
	published_at DATETIME(3) NULL COMMENT '发布时间',
	created_by VARCHAR(64) NOT NULL DEFAULT '' COMMENT '员工 ID',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	updated_at DATETIME(3) NOT NULL COMMENT '更新时间',
	UNIQUE KEY uk_app_releases_dim (product_code, channel_code, platform_code, arch_code, version_label),
	KEY idx_app_releases_latest (product_code, channel_code, platform_code, arch_code, release_status, published_at),
	KEY idx_app_releases_status (release_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用发布记录';
