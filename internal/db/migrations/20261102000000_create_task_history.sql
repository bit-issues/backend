-- +goose Up
-- +goose StatementBegin
CREATE TABLE `task_history` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `task_id` BIGINT UNSIGNED NOT NULL,
    `actor_id` BIGINT UNSIGNED NOT NULL,
    `action` ENUM('created', 'updated') NOT NULL,
    `changes` JSON NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_task_history_task_created` (`task_id`, `created_at`, `id`),
    KEY `idx_task_history_actor_id` (`actor_id`),
    CONSTRAINT `fk_task_history_task` FOREIGN KEY (`task_id`) REFERENCES `tasks`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_task_history_actor` FOREIGN KEY (`actor_id`) REFERENCES `users`(`id`) ON DELETE RESTRICT
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO `task_history` (`task_id`, `actor_id`, `action`, `changes`, `created_at`)
SELECT `id`, `author_id`, 'created', '[]', `created_at`
FROM `tasks`;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `task_history`;
-- +goose StatementEnd
