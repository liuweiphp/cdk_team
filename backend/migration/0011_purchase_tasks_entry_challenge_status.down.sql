ALTER TABLE `purchase_tasks`
    MODIFY COLUMN `status` ENUM(
        'pending',
        'registering',
        'ordering',
        'pending_payment',
        'fetching_subscribe',
        'ready',
        'needs_manual_review',
        'manual_completed',
        'failed'
    ) NOT NULL DEFAULT 'pending' COMMENT '采购任务状态';
