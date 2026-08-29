-- 001_init：niuma-cloud 初始表结构（SQLite，仅本地临时调试）
-- 约定与 MySQL 版对齐：
--   1) 列名禁止 SQL 关键字（status/body/key/user/order 等）
--   2) 时间存 UTC ISO8601 文本，语义对齐 MySQL DATETIME(3)
--   3) 表/列含义见行内注释（SQLite 无原生 COLUMN COMMENT）

-- Schema 迁移记录
CREATE TABLE IF NOT EXISTS schema_migrations (
	migration_id TEXT PRIMARY KEY, -- 迁移版本号，如 001_init
	applied_at TEXT NOT NULL       -- 应用时间 UTC
);

-- 终端用户
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,                    -- 用户 ID
	email TEXT NOT NULL UNIQUE,             -- 登录邮箱
	password_hash TEXT NOT NULL,            -- 密码哈希
	display_name TEXT NOT NULL DEFAULT '',  -- 显示名
	email_verified_at TEXT,                 -- 邮箱验证完成时间
	account_status TEXT NOT NULL DEFAULT 'active', -- 账户状态：active/disabled
	created_at TEXT NOT NULL,               -- 创建时间
	updated_at TEXT NOT NULL                -- 更新时间
);

-- 邮箱验证码
CREATE TABLE IF NOT EXISTS auth_otps (
	id TEXT PRIMARY KEY,           -- OTP 记录 ID
	email TEXT NOT NULL,           -- 目标邮箱
	purpose_code TEXT NOT NULL,    -- 用途：register/login/reset
	code_hash TEXT NOT NULL,       -- 验证码哈希
	expires_at TEXT NOT NULL,      -- 过期时间
	consumed_at TEXT,              -- 核销时间
	created_at TEXT NOT NULL       -- 创建时间
);

CREATE INDEX IF NOT EXISTS idx_auth_otps_email_purpose ON auth_otps(email, purpose_code);

-- 用户 Refresh 会话
CREATE TABLE IF NOT EXISTS refresh_sessions (
	id TEXT PRIMARY KEY,                         -- 会话 ID
	user_id TEXT NOT NULL,                       -- 用户 ID
	token_hash TEXT NOT NULL UNIQUE,              -- Refresh Token 哈希
	device_label TEXT NOT NULL DEFAULT '',        -- 设备标签
	expires_at TEXT NOT NULL,                     -- 过期时间
	revoked_at TEXT,                             -- 吊销时间
	created_at TEXT NOT NULL                      -- 创建时间
);

CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_sessions(user_id);

-- 用户反馈工单（对外 JSON 仍用 title/body/status，库内专业列名）
CREATE TABLE IF NOT EXISTS feedback (
	id TEXT PRIMARY KEY,                              -- 工单 ID
	reporter_user_id TEXT NOT NULL DEFAULT '',        -- 提交用户 ID，匿名为空
	category_code TEXT NOT NULL DEFAULT '',           -- 分类：bug/feature/other
	subject TEXT NOT NULL,                             -- 标题
	content_text TEXT NOT NULL,                        -- 正文
	contact_email TEXT NOT NULL DEFAULT '',           -- 联系邮箱（官网 lookup / 匿名通知）
	product_code TEXT NOT NULL DEFAULT '',            -- 产品标识
	client_version TEXT NOT NULL DEFAULT '',          -- 客户端版本
	ticket_status TEXT NOT NULL DEFAULT 'open',       -- open/triaged/closed
	internal_note TEXT NOT NULL DEFAULT '',           -- 内部备注，不对用户展示
	staff_reply TEXT NOT NULL DEFAULT '',             -- 对用户可见回复
	staff_reply_at TEXT,                              -- 回复时间
	staff_replier_id TEXT NOT NULL DEFAULT '',        -- 回复员工 ID
	reply_notified_at TEXT,                           -- 匿名回复邮件发送时间
	source_ip_hash TEXT NOT NULL DEFAULT '',          -- 来源 IP 哈希
	created_at TEXT NOT NULL,                          -- 创建时间
	updated_at TEXT NOT NULL                           -- 更新时间
);

CREATE INDEX IF NOT EXISTS idx_feedback_reporter ON feedback(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created ON feedback(created_at);
CREATE INDEX IF NOT EXISTS idx_feedback_ticket_status ON feedback(ticket_status);
CREATE INDEX IF NOT EXISTS idx_feedback_contact ON feedback(contact_email);

-- 运营员工
CREATE TABLE IF NOT EXISTS staff_users (
	id TEXT PRIMARY KEY,                              -- 员工 ID
	email TEXT NOT NULL UNIQUE,                       -- 登录邮箱
	password_hash TEXT NOT NULL,                      -- 密码哈希
	display_name TEXT NOT NULL DEFAULT '',            -- 显示名
	account_status TEXT NOT NULL DEFAULT 'active',    -- active/disabled
	created_at TEXT NOT NULL,                         -- 创建时间
	updated_at TEXT NOT NULL                          -- 更新时间
);

-- 员工会话（预留刷新/吊销；当前 Ops 以 JWT 为主）
CREATE TABLE IF NOT EXISTS staff_sessions (
	id TEXT PRIMARY KEY,                 -- 员工会话 ID
	staff_id TEXT NOT NULL,              -- 员工 ID
	token_hash TEXT NOT NULL UNIQUE,      -- 会话 Token 哈希
	expires_at TEXT NOT NULL,             -- 过期时间
	revoked_at TEXT,                     -- 吊销时间
	created_at TEXT NOT NULL              -- 创建时间
);

CREATE INDEX IF NOT EXISTS idx_staff_sessions_staff ON staff_sessions(staff_id);

-- 审计日志（关键业务动作：注册/登录/改密/反馈/运营处理等）
CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,                       -- 审计日志 ID
	actor_kind TEXT NOT NULL,                  -- user/staff/guest/system
	actor_id TEXT NOT NULL DEFAULT '',         -- 操作者 ID
	action_code TEXT NOT NULL,                 -- 如 auth.register / feedback.update
	resource_kind TEXT NOT NULL DEFAULT '',    -- 资源类型
	resource_id TEXT NOT NULL DEFAULT '',      -- 资源 ID
	detail_json TEXT NOT NULL DEFAULT '',      -- 详情 JSON
	source_ip_hash TEXT NOT NULL DEFAULT '',   -- 来源 IP 哈希
	created_at TEXT NOT NULL                    -- 创建时间
);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_kind, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_kind, actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action_code);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
