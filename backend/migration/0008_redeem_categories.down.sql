ALTER TABLE `redeem_items`
    DROP FOREIGN KEY `fk_redeem_items_category`,
    DROP INDEX `idx_redeem_items_category_created`,
    DROP COLUMN `category_id`;

DROP TABLE IF EXISTS `redeem_categories`;
