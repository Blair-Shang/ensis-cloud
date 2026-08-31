-- 006_ai_usage：系统模型按用户按日计量（SQLite）

CREATE TABLE IF NOT EXISTS ai_usage_daily (
	user_id TEXT NOT NULL,       -- 终端用户 ID
	usage_day TEXT NOT NULL,     -- UTC YYYY-MM-DD
	request_count INTEGER NOT NULL DEFAULT 0, -- 当日请求次数
	token_count INTEGER NOT NULL DEFAULT 0,   -- 当日累计 token
	updated_at TEXT NOT NULL,    -- 最近更新时间
	PRIMARY KEY (user_id, usage_day)
);

CREATE INDEX IF NOT EXISTS idx_ai_usage_day ON ai_usage_daily (usage_day);
