-- 006_ai_usage：系统模型按用户按日计量（请求次数 / 估算 token）

CREATE TABLE IF NOT EXISTS ai_usage_daily (
	user_id VARCHAR(64) NOT NULL COMMENT '终端用户 ID',
	usage_day DATE NOT NULL COMMENT 'UTC 自然日',
	request_count INT NOT NULL DEFAULT 0 COMMENT '当日请求次数',
	token_count INT NOT NULL DEFAULT 0 COMMENT '当日累计 token（有则记）',
	updated_at DATETIME(3) NOT NULL COMMENT '最近更新时间 UTC',
	PRIMARY KEY (user_id, usage_day),
	KEY idx_ai_usage_day (usage_day)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统 AI 按日用量';
