-- 反馈截图 / 附件元数据（文件落盘 data_dir/feedback/{id}/）
ALTER TABLE feedback
	ADD COLUMN attachments_json JSON NULL COMMENT '附件元数据 JSON 数组' AFTER source_ip_hash;

UPDATE feedback
SET attachments_json = JSON_ARRAY()
WHERE attachments_json IS NULL;
