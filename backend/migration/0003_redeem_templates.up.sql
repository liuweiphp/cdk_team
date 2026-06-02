CREATE TABLE `redeem_templates` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(128) NOT NULL COMMENT '模板名称',
    `content` TEXT NOT NULL COMMENT '模板内容,使用 {{content}} 作为兑换内容占位符',
    `status` ENUM('active','disabled') NOT NULL DEFAULT 'active' COMMENT '状态: active=可用, disabled=禁用',
    `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建管理员ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    INDEX `idx_status_created` (`status`, `created_at`),
    INDEX `idx_deleted` (`deleted_at`),
    INDEX `idx_created_by` (`created_by`),
    CONSTRAINT `fk_redeem_templates_user` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='兑换模板表';

ALTER TABLE `redeem_items`
    ADD COLUMN `template_id` BIGINT UNSIGNED NULL COMMENT '生成该兑换内容使用的模板ID' AFTER `content`,
    ADD INDEX `idx_template_created` (`template_id`, `created_at`),
    ADD CONSTRAINT `fk_redeem_items_template` FOREIGN KEY (`template_id`) REFERENCES `redeem_templates` (`id`);

INSERT INTO `redeem_templates` (`name`, `content`, `status`, `created_by`) VALUES
('默认账号模板', '配 置 链 接↓ ↓ ↓ ↓ ↓ ↓↓ ↓ ↓ ↓ ↓ ↓↓ ↓ ↓ ↓ ↓ ↓
{{content}}

有其他账号需求的话，可以咨询。
ID成品号（永久）￥46
chatgpt plus充值会员￥60
tiktok账号￥48
推特新号￥36
推特老号￥66
飞机新号￥36
飞机老号（稳定）￥66
谷歌账号￥36
谷歌老号￥66
ins账号￥25
脸书账号￥25

免费收徒
免费收徒
免费收徒
免费收徒', 'active', 1);
