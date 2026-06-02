DROP TABLE IF EXISTS `purchase_tasks`;
DROP TABLE IF EXISTS `team_template_sequences`;

ALTER TABLE `redeem_templates`
    DROP COLUMN `result_content_mode`,
    DROP COLUMN `external_provider`,
    DROP COLUMN `external_target_name`,
    DROP COLUMN `external_target_code`;

ALTER TABLE `users`
    DROP COLUMN `external_account_prefix`;
