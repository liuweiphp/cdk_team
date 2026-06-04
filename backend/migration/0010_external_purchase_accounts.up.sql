CREATE TABLE IF NOT EXISTS `external_account_sequences` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `provider` VARCHAR(64) NOT NULL DEFAULT 'yfjc' COMMENT '外部提供方',
    `current_seq` BIGINT UNSIGNED NOT NULL DEFAULT 1000 COMMENT '当前已分配账号序号',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_external_account_sequence_provider` (`provider`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='外部采购账号序号表';

ALTER TABLE `purchase_tasks`
    ADD COLUMN `external_account_seq` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '外部采购账号序号' AFTER `provider`,
    ADD COLUMN `external_username` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '外部采购账号用户名' AFTER `external_account_seq`,
    ADD COLUMN `external_password` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '外部采购账号密码' AFTER `external_username`;
