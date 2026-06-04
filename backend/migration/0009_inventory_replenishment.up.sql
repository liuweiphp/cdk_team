ALTER TABLE `redeem_templates`
    ADD COLUMN `safe_stock` INT NOT NULL DEFAULT 0 COMMENT '安全库存阈值' AFTER `result_content_mode`,
    ADD COLUMN `replenish_quantity` INT NOT NULL DEFAULT 1 COMMENT '单次补货数量' AFTER `safe_stock`,
    ADD COLUMN `auto_replenish` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否自动补货' AFTER `replenish_quantity`;

ALTER TABLE `purchase_tasks`
    ADD COLUMN `source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT '任务来源: manual/replenishment' AFTER `provider`,
    ADD INDEX `idx_purchase_source_created` (`source`, `created_at`);
