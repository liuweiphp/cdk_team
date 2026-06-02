DELETE FROM `cdks`
WHERE `import_id` IN (
    SELECT `id` FROM `cdk_imports` WHERE `filename` = 'backfill_redeem_items'
);

DELETE FROM `cdk_imports`
WHERE `filename` = 'backfill_redeem_items';
