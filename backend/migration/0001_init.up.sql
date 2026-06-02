-- CDK 兑换系统初始化迁移

CREATE TABLE `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username` VARCHAR(64) NOT NULL COMMENT '用户名',
    `password_hash` VARCHAR(255) NOT NULL COMMENT 'bcrypt 密码哈希',
    `role` ENUM('admin','user') NOT NULL DEFAULT 'user' COMMENT '角色: admin=管理员, user=普通用户',
    `status` ENUM('active','disabled') NOT NULL DEFAULT 'active' COMMENT '状态: active=正常, disabled=已禁用',
    `last_login_at` TIMESTAMP NULL DEFAULT NULL COMMENT '最后登录时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_username` (`username`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

CREATE TABLE `cdk_imports` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `filename` VARCHAR(255) NOT NULL COMMENT '上传原始文件名',
    `amount` DECIMAL(12,2) NOT NULL COMMENT '本次导入的金额(面额)',
    `total` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文件总行数',
    `inserted` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '实际入库条数',
    `skipped` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '重复跳过条数',
    `invalid` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '格式无效条数',
    `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '导入操作的管理员用户ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '导入时间',
    PRIMARY KEY (`id`),
    INDEX `idx_amount_created` (`amount`, `created_at`),
    INDEX `idx_created_by` (`created_by`),
    CONSTRAINT `fk_imports_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CDK导入记录表';

CREATE TABLE `cdks` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `code` VARCHAR(64) NOT NULL COMMENT 'CDK码,统一大写,一码一兑',
    `amount` DECIMAL(12,2) NOT NULL COMMENT '面额,继承自导入时的设定',
    `status` ENUM('unused','exchanged') NOT NULL DEFAULT 'unused' COMMENT '状态: unused=未领取, exchanged=已领取',
    `import_id` BIGINT UNSIGNED NOT NULL COMMENT '所属导入批次ID,溯源用',
    `exchanged_by` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '领取用户ID',
    `exchanged_at` TIMESTAMP NULL DEFAULT NULL COMMENT '领取时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_code` (`code`),
    INDEX `idx_amount_status` (`amount`, `status`),
    INDEX `idx_status_exchanged` (`status`, `exchanged_at`),
    INDEX `idx_exchanged_by` (`exchanged_by`, `exchanged_at`),
    CONSTRAINT `fk_cdks_import` FOREIGN KEY (`import_id`) REFERENCES `cdk_imports` (`id`),
    CONSTRAINT `fk_cdks_user` FOREIGN KEY (`exchanged_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='兑换码表';

CREATE TABLE `exchange_orders` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '领取用户ID',
    `amount` DECIMAL(12,2) NOT NULL COMMENT '领取的面额',
    `quantity` INT UNSIGNED NOT NULL COMMENT '本次领取张数',
    `total_amount` DECIMAL(14,2) NOT NULL COMMENT '总金额 = amount * quantity',
    `status` ENUM('success','failed') NOT NULL COMMENT '状态: success=成功, failed=失败',
    `fail_reason` VARCHAR(64) NULL DEFAULT NULL COMMENT '失败原因: insufficient_stock / rate_limited / invalid_input',
    `ip` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '客户端IP',
    `user_agent` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '客户端User-Agent',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_user_created` (`user_id`, `created_at`),
    INDEX `idx_amount_status_created` (`amount`, `status`, `created_at`),
    INDEX `idx_status_created` (`status`, `created_at`),
    CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='领取订单表';

CREATE TABLE `exchange_order_items` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
    `cdk_id` BIGINT UNSIGNED NOT NULL COMMENT '领取的CDK ID',
    `code` VARCHAR(64) NOT NULL COMMENT '冗余CDK码,便于查询不join',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_order_id` (`order_id`),
    UNIQUE INDEX `idx_cdk_id` (`cdk_id`),
    CONSTRAINT `fk_items_order` FOREIGN KEY (`order_id`) REFERENCES `exchange_orders` (`id`),
    CONSTRAINT `fk_items_cdk` FOREIGN KEY (`cdk_id`) REFERENCES `cdks` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='领取明细表';

CREATE TABLE `announcements` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `title` VARCHAR(255) NOT NULL COMMENT '公告标题',
    `content` TEXT NOT NULL COMMENT '富文本HTML内容,入库前已净化',
    `is_pinned` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否置顶',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '发布管理员ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发布时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    INDEX `idx_pinned_created` (`is_pinned`, `created_at`),
    INDEX `idx_deleted` (`deleted_at`),
    CONSTRAINT `fk_announcements_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公告表';

-- 插入默认管理员 (密码: admin123, bcrypt cost=12)
-- 实际环境中请立即修改密码
INSERT INTO `users` (`username`, `password_hash`, `role`) VALUES
('admin', '$2a$12$BvANL.DWudIHw71xt82kjeanAX7DV6.biRj8R6kYpEyEYkuDYWDOK', 'admin');
