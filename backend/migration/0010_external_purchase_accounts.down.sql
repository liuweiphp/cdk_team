ALTER TABLE `purchase_tasks`
    DROP COLUMN `external_password`,
    DROP COLUMN `external_username`,
    DROP COLUMN `external_account_seq`;

DROP TABLE IF EXISTS `external_account_sequences`;
