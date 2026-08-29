-- 001_init：niuma-cloud 初始表结构（MySQL）
-- 约定：
--   1) 时间字段统一 DATETIME(3)，存 UTC（DSN loc=UTC + parseTime=true）
--   2) 禁止使用 SQL 关键字作列名（如 status/body/key/user/order 等）
--   3) 业务表列均带 COMMENT

CREATE TABLE IF NOT EXISTS schema_migrations (
	migration_id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '迁移版本号，如 001_init',
	applied_at DATETIME(3) NOT NULL COMMENT '应用时间 UTC'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Schema 迁移记录';

CREATE TABLE IF NOT EXISTS users (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '用户 ID',
	email VARCHAR(320) NOT NULL COMMENT '登录邮箱',
	password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
	display_name VARCHAR(128) NOT NULL DEFAULT '' COMMENT '显示名',
	email_verified_at DATETIME(3) NULL COMMENT '邮箱验证完成时间',
	account_status VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '账户状态：active/disabled',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	updated_at DATETIME(3) NOT NULL COMMENT '更新时间',
	UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='终端用户';

CREATE TABLE IF NOT EXISTS auth_otps (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT 'OTP 记录 ID',
	email VARCHAR(320) NOT NULL COMMENT '目标邮箱',
	purpose_code VARCHAR(32) NOT NULL COMMENT '用途：register/login/reset',
	code_hash VARCHAR(64) NOT NULL COMMENT '验证码哈希',
	expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
	consumed_at DATETIME(3) NULL COMMENT '核销时间',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	KEY idx_auth_otps_email_purpose (email, purpose_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邮箱验证码';

CREATE TABLE IF NOT EXISTS refresh_sessions (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '会话 ID',
	user_id VARCHAR(64) NOT NULL COMMENT '用户 ID',
	token_hash VARCHAR(64) NOT NULL COMMENT 'Refresh Token 哈希',
	device_label VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备标签',
	expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
	revoked_at DATETIME(3) NULL COMMENT '吊销时间',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	UNIQUE KEY uk_refresh_token (token_hash),
	KEY idx_refresh_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户 Refresh 会话';

-- 反馈工单：对外 JSON 仍用 title/body/status 等固定字段，库内用专业列名映射
CREATE TABLE IF NOT EXISTS feedback (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '工单 ID',
	reporter_user_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '提交用户 ID，匿名为空',
	category_code VARCHAR(64) NOT NULL DEFAULT '' COMMENT '分类：bug/feature/other',
	subject VARCHAR(256) NOT NULL COMMENT '标题',
	content_text TEXT NOT NULL COMMENT '正文',
	contact_email VARCHAR(320) NOT NULL DEFAULT '' COMMENT '联系邮箱（回复通知目标）',
	product_code VARCHAR(64) NOT NULL DEFAULT '' COMMENT '产品标识',
	client_version VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端版本',
	ticket_status VARCHAR(32) NOT NULL DEFAULT 'open' COMMENT '工单状态：open/triaged/closed',
	internal_note TEXT NOT NULL COMMENT '内部备注，不对用户展示',
	staff_reply TEXT NOT NULL COMMENT '对用户可见的回复内容',
	staff_reply_at DATETIME(3) NULL COMMENT '回复时间',
	staff_replier_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '回复员工 ID',
	reply_notified_at DATETIME(3) NULL COMMENT '回复邮件通知发送时间',
	source_ip_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT '来源 IP 哈希',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	updated_at DATETIME(3) NOT NULL COMMENT '更新时间',
	KEY idx_feedback_reporter (reporter_user_id),
	KEY idx_feedback_created (created_at),
	KEY idx_feedback_ticket_status (ticket_status),
	KEY idx_feedback_contact (contact_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户反馈工单';

CREATE TABLE IF NOT EXISTS staff_users (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '员工 ID',
	email VARCHAR(320) NOT NULL COMMENT '登录邮箱',
	password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
	display_name VARCHAR(128) NOT NULL DEFAULT '' COMMENT '显示名',
	account_status VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '账户状态：active/disabled',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	updated_at DATETIME(3) NOT NULL COMMENT '更新时间',
	UNIQUE KEY uk_staff_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营员工';

CREATE TABLE IF NOT EXISTS staff_sessions (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '员工会话 ID',
	staff_id VARCHAR(64) NOT NULL COMMENT '员工 ID',
	token_hash VARCHAR(64) NOT NULL COMMENT '会话 Token 哈希',
	expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
	revoked_at DATETIME(3) NULL COMMENT '吊销时间',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	UNIQUE KEY uk_staff_token (token_hash),
	KEY idx_staff_sessions_staff (staff_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='员工会话';

CREATE TABLE IF NOT EXISTS audit_logs (
	id VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '审计日志 ID',
	actor_kind VARCHAR(32) NOT NULL COMMENT '操作者类型：user/staff/system',
	actor_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作者 ID',
	action_code VARCHAR(64) NOT NULL COMMENT '动作编码，如 feedback.reply',
	resource_kind VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源类型',
	resource_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源 ID',
	detail_json TEXT NOT NULL COMMENT '详情 JSON',
	source_ip_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT '来源 IP 哈希',
	created_at DATETIME(3) NOT NULL COMMENT '创建时间',
	KEY idx_audit_resource (resource_kind, resource_id),
	KEY idx_audit_actor (actor_kind, actor_id),
	KEY idx_audit_action (action_code),
	KEY idx_audit_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审计日志';
