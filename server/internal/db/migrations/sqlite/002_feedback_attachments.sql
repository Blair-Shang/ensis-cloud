-- 反馈截图 / 附件元数据（文件落盘 data_dir/feedback/{id}/）
ALTER TABLE feedback ADD COLUMN attachments_json TEXT NOT NULL DEFAULT '[]';
