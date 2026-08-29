-- 005_staff_role：员工角色（admin 超管 / operator 日常运营）
-- 已有员工默认 admin，避免升级后锁死当前账号

ALTER TABLE staff_users
	ADD COLUMN staff_role VARCHAR(32) NOT NULL DEFAULT 'admin' COMMENT '角色：admin/operator' AFTER account_status;
