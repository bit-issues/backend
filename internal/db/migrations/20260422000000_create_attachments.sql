-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `attachments` (
    `id` SERIAL,
    `task_id` BIGINT UNSIGNED NOT NULL,
    `file_name` VARCHAR(255) NOT NULL,
    `storage_key` VARCHAR(512) NOT NULL,
    `size_bytes` BIGINT UNSIGNED NOT NULL,
    `status` ENUM('pending', 'uploaded') NOT NULL DEFAULT 'pending',
    `uploaded_by` BIGINT UNSIGNED NOT NULL,
    `uploaded_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL,
    PRIMARY KEY (`id`),
    INDEX `idx_attachments_task_id` (`task_id`),
    INDEX `idx_attachments_storage_key` (`storage_key`),
    INDEX `idx_attachments_status` (`status`),
    INDEX `idx_attachments_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_attachments_task` FOREIGN KEY (`task_id`) REFERENCES `tasks`(`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_attachments_uploader` FOREIGN KEY (`uploaded_by`) REFERENCES `users`(`id`) ON DELETE RESTRICT
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `attachments`;
-- +goose StatementEnd