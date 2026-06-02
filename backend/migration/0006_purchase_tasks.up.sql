ALTER TABLE `users`
    ADD COLUMN `external_account_prefix` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '外部账号前缀,用于拼接采购账号标识' AFTER `status`;

ALTER TABLE `redeem_templates`
    ADD COLUMN `external_target_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '模板对应外部目标编码' AFTER `content`,
    ADD COLUMN `external_target_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '模板对应外部目标名称' AFTER `external_target_code`,
    ADD COLUMN `external_provider` VARCHAR(64) NOT NULL DEFAULT 'yfjc' COMMENT '模板对应外部提供方' AFTER `external_target_name`,
    ADD COLUMN `result_content_mode` VARCHAR(32) NOT NULL DEFAULT 'subscribe_url' COMMENT '结果内容模式' AFTER `external_provider`;

CREATE TABLE `team_template_sequences` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `team_owner_id` BIGINT UNSIGNED NOT NULL COMMENT '团队拥有者用户ID',
    `template_id` BIGINT UNSIGNED NOT NULL COMMENT '模板ID',
    `current_seq` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '当前已分配序号',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_team_template_sequence` (`team_owner_id`, `template_id`),
    CONSTRAINT `fk_team_template_sequences_owner` FOREIGN KEY (`team_owner_id`) REFERENCES `users` (`id`),
    CONSTRAINT `fk_team_template_sequences_template` FOREIGN KEY (`template_id`) REFERENCES `redeem_templates` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='团队模板序号表';

CREATE TABLE `purchase_tasks` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `team_owner_id` BIGINT UNSIGNED NOT NULL COMMENT '团队拥有者用户ID',
    `template_id` BIGINT UNSIGNED NOT NULL COMMENT '模板ID',
    `redeem_item_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '生成的兑换内容ID',
    `cdk_id` BIGINT UNSIGNED NOT NULL COMMENT '关联CDK ID',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人用户ID',
    `account_prefix` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '账号前缀',
    `account_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '账号名称',
    `template_code_part` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '模板编码片段',
    `sequence_no` BIGINT UNSIGNED NOT NULL COMMENT '模板内递增序号',
    `target_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '目标编码',
    `target_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '目标名称',
    `provider` VARCHAR(64) NOT NULL DEFAULT 'yfjc' COMMENT '提供方',
    `status` ENUM('pending','registering','ordering','pending_payment','fetching_subscribe','ready','needs_manual_review','manual_completed','failed') NOT NULL DEFAULT 'pending' COMMENT '采购任务状态',
    `retry_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '重试次数',
    `payment_status` ENUM('unpaid','paid','unknown') NOT NULL DEFAULT 'unpaid' COMMENT '支付状态',
    `manual_review_reason` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '人工复核原因',
    `external_order_no` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '外部订单号',
    `subscribe_url` TEXT NULL COMMENT '订阅链接',
    `last_error` TEXT NULL COMMENT '最近一次错误信息',
    `browser_trace_path` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '浏览器追踪文件路径',
    `screenshot_path` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '截图文件路径',
    `html_dump_path` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'HTML dump 文件路径',
    `payload_json` LONGTEXT NULL COMMENT '外部请求负载',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_purchase_owner_template_seq` (`team_owner_id`, `template_id`, `sequence_no`),
    UNIQUE INDEX `idx_purchase_redeem_item` (`redeem_item_id`),
    UNIQUE INDEX `idx_purchase_cdk` (`cdk_id`),
    INDEX `idx_purchase_owner_status_created` (`team_owner_id`, `status`, `created_at`),
    INDEX `idx_purchase_template_status_created` (`template_id`, `status`, `created_at`),
    INDEX `idx_purchase_deleted` (`deleted_at`),
    INDEX `idx_purchase_created_by` (`created_by`),
    CONSTRAINT `fk_purchase_tasks_owner` FOREIGN KEY (`team_owner_id`) REFERENCES `users` (`id`),
    CONSTRAINT `fk_purchase_tasks_template` FOREIGN KEY (`template_id`) REFERENCES `redeem_templates` (`id`),
    CONSTRAINT `fk_purchase_tasks_redeem_item` FOREIGN KEY (`redeem_item_id`) REFERENCES `redeem_items` (`id`),
    CONSTRAINT `fk_purchase_tasks_cdk` FOREIGN KEY (`cdk_id`) REFERENCES `cdks` (`id`),
    CONSTRAINT `fk_purchase_tasks_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='采购任务表';
