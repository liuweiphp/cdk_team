CREATE TABLE `teams` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `owner_id` BIGINT UNSIGNED NOT NULL COMMENT '团队拥有者用户ID',
    `name` VARCHAR(128) NOT NULL COMMENT '团队名称',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_teams_owner` (`owner_id`),
    INDEX `idx_teams_deleted` (`deleted_at`),
    CONSTRAINT `fk_teams_owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='团队表';

CREATE TABLE `team_members` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `team_id` BIGINT UNSIGNED NOT NULL COMMENT '团队ID',
    `member_id` BIGINT UNSIGNED NOT NULL COMMENT '成员用户ID',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_team_member` (`team_id`, `member_id`),
    INDEX `idx_member_id` (`member_id`),
    CONSTRAINT `fk_team_members_team` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`),
    CONSTRAINT `fk_team_members_user` FOREIGN KEY (`member_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='团队成员表';

INSERT INTO `teams` (`owner_id`, `name`)
SELECT `id`, CONCAT(`username`, '的团队')
FROM `users`
WHERE `deleted_at` IS NULL;

INSERT INTO `team_members` (`team_id`, `member_id`)
SELECT `teams`.`id`, `teams`.`owner_id`
FROM `teams`;
