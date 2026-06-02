CREATE TABLE `redeem_categories` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(128) NOT NULL COMMENT '兑换内容分类名称',
    `status` ENUM('active','disabled') NOT NULL DEFAULT 'active' COMMENT '状态: active=启用, disabled=禁用',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建管理员ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    INDEX `idx_redeem_categories_status_created` (`status`, `created_at`),
    INDEX `idx_redeem_categories_deleted` (`deleted_at`),
    INDEX `idx_redeem_categories_created_by` (`created_by`),
    CONSTRAINT `fk_redeem_categories_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='兑换内容分类表';

ALTER TABLE `redeem_items`
    ADD COLUMN `category_id` BIGINT UNSIGNED NULL COMMENT '兑换内容分类ID' AFTER `content`,
    ADD INDEX `idx_redeem_items_category_created` (`category_id`, `created_at`),
    ADD CONSTRAINT `fk_redeem_items_category` FOREIGN KEY (`category_id`) REFERENCES `redeem_categories` (`id`);
