-- 007_ai_manage：系统模型接入点 / 模型目录走库表；用量表补齐审计字段（SQLite）

CREATE TABLE IF NOT EXISTS ai_upstream_endpoint (
	endpoint_id TEXT NOT NULL PRIMARY KEY,          -- 接入点 ID
	provider_display_name TEXT NOT NULL,            -- 桌面展示名
	upstream_base_url TEXT NOT NULL,                -- OpenAI 兼容根地址
	api_secret_cipher TEXT NOT NULL,                -- 上游 API 密钥密文
	service_enabled INTEGER NOT NULL DEFAULT 0,     -- 1 开放 / 0 关闭
	default_model_code TEXT NOT NULL DEFAULT '',    -- 默认模型编码
	daily_request_limit INTEGER NOT NULL DEFAULT 80,
	daily_token_limit INTEGER NOT NULL DEFAULT 0,
	sort_seq INTEGER NOT NULL DEFAULT 0,            -- 展示顺序
	created_by_staff_id TEXT NOT NULL DEFAULT '',   -- 新增人
	updated_by_staff_id TEXT NOT NULL DEFAULT '',   -- 编辑人
	created_at TEXT NOT NULL,                       -- 新增时间
	updated_at TEXT NOT NULL                        -- 编辑时间
);

CREATE TABLE IF NOT EXISTS ai_model_catalog (
	catalog_id TEXT NOT NULL PRIMARY KEY,           -- 目录条目 ID
	endpoint_id TEXT NOT NULL,                      -- 所属接入点 ID
	model_code TEXT NOT NULL,                       -- 上游模型编码
	model_label TEXT NOT NULL DEFAULT '',           -- 展示名
	record_status TEXT NOT NULL DEFAULT 'active',   -- active/disabled
	sort_seq INTEGER NOT NULL DEFAULT 0,            -- 展示顺序
	created_by_staff_id TEXT NOT NULL DEFAULT '',   -- 新增人
	updated_by_staff_id TEXT NOT NULL DEFAULT '',   -- 编辑人
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE (endpoint_id, model_code)
);

CREATE INDEX IF NOT EXISTS idx_ai_endpoint_enabled ON ai_upstream_endpoint (service_enabled, sort_seq);
CREATE INDEX IF NOT EXISTS idx_ai_model_endpoint ON ai_model_catalog (endpoint_id, record_status, sort_seq);

ALTER TABLE ai_usage_daily ADD COLUMN created_by_staff_id TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_usage_daily ADD COLUMN updated_by_staff_id TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_usage_daily ADD COLUMN created_at TEXT;
