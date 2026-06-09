ALTER TABLE `users`
    ADD COLUMN `file_prefix` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '批量生成文件名前缀' AFTER `status`,
    ADD COLUMN `file_sequence_next` INT UNSIGNED NOT NULL DEFAULT 1001 COMMENT '下一个批量生成文件序号' AFTER `file_prefix`;
