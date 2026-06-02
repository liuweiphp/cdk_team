CREATE TABLE `redeem_items` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(128) NOT NULL COMMENT '兑换内容名称',
    `filename` VARCHAR(255) NOT NULL COMMENT '兑换后下载的文本文件名',
    `content` TEXT NOT NULL COMMENT '文本文件内容',
    `status` ENUM('active','disabled') NOT NULL DEFAULT 'active' COMMENT '状态: active=可兑换, disabled=禁用',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建管理员ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    INDEX `idx_status_created` (`status`, `created_at`),
    INDEX `idx_deleted` (`deleted_at`),
    INDEX `idx_created_by` (`created_by`),
    CONSTRAINT `fk_redeem_items_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='兑换内容表';

ALTER TABLE `cdk_imports`
    ADD COLUMN `item_id` BIGINT UNSIGNED NULL COMMENT '本次导入绑定的兑换内容ID' AFTER `amount`,
    ADD INDEX `idx_imports_item_created` (`item_id`, `created_at`),
    ADD CONSTRAINT `fk_imports_redeem_item` FOREIGN KEY (`item_id`) REFERENCES `redeem_items` (`id`);

ALTER TABLE `cdks`
    ADD COLUMN `item_id` BIGINT UNSIGNED NULL COMMENT '可兑换内容ID' AFTER `amount`,
    ADD INDEX `idx_item_status` (`item_id`, `status`),
    ADD CONSTRAINT `fk_cdks_redeem_item` FOREIGN KEY (`item_id`) REFERENCES `redeem_items` (`id`);
