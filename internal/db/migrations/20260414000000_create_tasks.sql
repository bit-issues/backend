-- +goose Up
-- +goose StatementBegin
CREATE TABLE `tasks` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_slug` VARCHAR(255) NOT NULL,
    `number` INT NOT NULL,
    `title` VARCHAR(255) NOT NULL,
    `description` TEXT,
    `priority` ENUM(
        'Trivial',
        'Minor',
        'Major',
        'Critical',
        'Blocker'
    ) NOT NULL DEFAULT 'Minor',
    `status` ENUM(
        'New',
        'Open',
        'In Progress',
        'Resolved',
        'Closed',
        'Reopened'
    ) NOT NULL DEFAULT 'New',
    `author_id` BIGINT UNSIGNED NOT NULL,
    `assignee_id` BIGINT UNSIGNED,
    `due_date` DATE,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tasks_project_number` (`project_slug`, `number`),
    KEY `idx_tasks_author_id` (`author_id`),
    KEY `idx_tasks_assignee_id` (`assignee_id`),
    KEY `idx_tasks_status_priority` (`status`, `priority`),
    KEY `idx_tasks_created_at` (`created_at`),
    KEY `idx_tasks_due_date` (`due_date`),
    KEY `idx_tasks_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_tasks_project` FOREIGN KEY (`project_slug`) REFERENCES `projects`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_tasks_author` FOREIGN KEY (`author_id`) REFERENCES `users`(`id`) ON DELETE RESTRICT,
    CONSTRAINT `fk_tasks_assignee` FOREIGN KEY (`assignee_id`) REFERENCES `users`(`id`) ON DELETE
    SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE `tasks`;
-- +goose StatementEnd