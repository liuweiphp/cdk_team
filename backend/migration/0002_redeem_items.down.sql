ALTER TABLE `cdks`
    DROP FOREIGN KEY `fk_cdks_redeem_item`,
    DROP INDEX `idx_item_status`,
    DROP COLUMN `item_id`;

ALTER TABLE `cdk_imports`
    DROP FOREIGN KEY `fk_imports_redeem_item`,
    DROP INDEX `idx_imports_item_created`,
    DROP COLUMN `item_id`;

DROP TABLE `redeem_items`;
