ALTER TABLE `purchase_tasks`
    DROP INDEX `idx_purchase_source_created`,
    DROP COLUMN `source`;

ALTER TABLE `redeem_templates`
    DROP COLUMN `auto_replenish`,
    DROP COLUMN `replenish_quantity`,
    DROP COLUMN `safe_stock`;
