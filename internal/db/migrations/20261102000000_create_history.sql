-- +goose Up
-- +goose StatementBegin
CREATE TABLE `history` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `subject_type` ENUM('task') NOT NULL,
    `subject_id` BIGINT UNSIGNED NOT NULL,
    `actor_id` BIGINT UNSIGNED NOT NULL,
    `action` ENUM(
        'created',
        'updated',
        'attachment_added',
        'attachment_removed'
    ) NOT NULL,
    `payload` JSON NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_history_subject_created` (`subject_type`, `subject_id`, `created_at`, `id`),
    KEY `idx_history_actor_id` (`actor_id`),
    CONSTRAINT `fk_history_actor` FOREIGN KEY (`actor_id`) REFERENCES `users`(`id`) ON DELETE RESTRICT
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `history`;
-- +goose StatementEnd