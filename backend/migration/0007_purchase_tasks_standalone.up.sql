ALTER TABLE `purchase_tasks`
    DROP FOREIGN KEY `fk_purchase_tasks_cdk`;

ALTER TABLE `purchase_tasks`
    MODIFY COLUMN `cdk_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '生成的CDK ID';

ALTER TABLE `purchase_tasks`
    ADD CONSTRAINT `fk_purchase_tasks_cdk` FOREIGN KEY (`cdk_id`) REFERENCES `cdks` (`id`);
