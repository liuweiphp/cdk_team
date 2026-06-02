INSERT INTO `cdk_imports` (`filename`, `amount`, `total`, `inserted`, `invalid`, `remark`, `created_by`)
SELECT 'backfill_redeem_items', 0, COUNT(*), COUNT(*), 0, '历史兑换内容自动补码', 1
FROM `redeem_items` ri
LEFT JOIN `cdks` c ON c.`item_id` = ri.`id`
WHERE c.`id` IS NULL;

INSERT INTO `cdks` (`code`, `amount`, `item_id`, `status`, `import_id`)
SELECT UPPER(SUBSTRING(REPLACE(UUID(), '-', ''), 1, 16)), 0, ri.`id`, 'unused', LAST_INSERT_ID()
FROM `redeem_items` ri
LEFT JOIN `cdks` c ON c.`item_id` = ri.`id`
WHERE c.`id` IS NULL;
