-- 007_ai_manage：系统模型接入点 / 模型目录走库表；用量表补齐审计字段
-- 约定：禁止 SQL 关键字列名；时间 UTC DATETIME(3)；新增人 / 编辑人为员工 ID

CREATE TABLE IF NOT EXISTS ai_upstream_endpoint (
	endpoint_id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '接入点 ID',
	provider_display_name VARCHAR(128) NOT NULL COMMENT '桌面展示名',
	upstream_base_url VARCHAR(512) NOT NULL COMMENT 'OpenAI 兼容根地址，通常到 /v1',
	api_secret_cipher TEXT NOT NULL COMMENT '上游 API 密钥密文，禁止明文列',
	service_enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否对终端用户开放：1 开放 / 0 关闭',
	default_model_code VARCHAR(128) NOT NULL DEFAULT '' COMMENT '默认模型编码',
	daily_request_limit INT NOT NULL DEFAULT 80 COMMENT '每用户每日请求上限，0 表示不限制',
	daily_token_limit INT NOT NULL DEFAULT 0 COMMENT '每用户每日 token 上限，0 表示不限制',
	sort_seq INT NOT NULL DEFAULT 0 COMMENT '展示顺序，数值小的优先',
	created_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '新增人（员工 ID）',
	updated_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '编辑人（员工 ID）',
	created_at DATETIME(3) NOT NULL COMMENT '新增时间 UTC',
	updated_at DATETIME(3) NOT NULL COMMENT '编辑时间 UTC',
	KEY idx_ai_endpoint_enabled (service_enabled, sort_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统模型上游接入点';

CREATE TABLE IF NOT EXISTS ai_model_catalog (
	catalog_id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '目录条目 ID',
	endpoint_id VARCHAR(64) NOT NULL COMMENT '所属接入点 ID',
	model_code VARCHAR(128) NOT NULL COMMENT '上游模型编码',
	model_label VARCHAR(128) NOT NULL DEFAULT '' COMMENT '展示名',
	record_status VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '记录状态：active/disabled',
	sort_seq INT NOT NULL DEFAULT 0 COMMENT '展示顺序，数值小的优先',
	created_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '新增人（员工 ID）',
	updated_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '编辑人（员工 ID）',
	created_at DATETIME(3) NOT NULL COMMENT '新增时间 UTC',
	updated_at DATETIME(3) NOT NULL COMMENT '编辑时间 UTC',
	UNIQUE KEY uk_ai_model_endpoint_code (endpoint_id, model_code),
	KEY idx_ai_model_endpoint (endpoint_id, record_status, sort_seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统模型目录';

ALTER TABLE ai_usage_daily
	ADD COLUMN created_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '新增人（系统计量为空）' AFTER token_count,
	ADD COLUMN updated_by_staff_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '编辑人（系统计量为空）' AFTER created_by_staff_id,
	ADD COLUMN created_at DATETIME(3) NULL COMMENT '首次计量时间 UTC' AFTER updated_by_staff_id;

UPDATE ai_usage_daily SET created_at = updated_at WHERE created_at IS NULL;
