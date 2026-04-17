-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `comments` (
    `id` SERIAL,
    `task_id` BIGINT UNSIGNED NOT NULL,
    `author_id` BIGINT UNSIGNED NOT NULL,
    `content` TEXT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL,
    PRIMARY KEY (`id`),
    INDEX `idx_comments_task_id` (`task_id`),
    INDEX `idx_comments_author_id` (`author_id`),
    INDEX `idx_comments_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_comments_task` FOREIGN KEY (`task_id`) REFERENCES `tasks`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_comments_author` FOREIGN KEY (`author_id`) REFERENCES `users`(`id`) ON DELETE RESTRICT
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `comments`;
-- +goose StatementEnd