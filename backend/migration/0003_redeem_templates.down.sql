ALTER TABLE `redeem_items`
    DROP FOREIGN KEY `fk_redeem_items_template`,
    DROP INDEX `idx_template_created`,
    DROP COLUMN `template_id`;

DROP TABLE `redeem_templates`;
