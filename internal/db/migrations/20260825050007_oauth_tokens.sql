-- +goose Up
-- +goose StatementBegin
CREATE TABLE `oauth_tokens` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `access_token` TEXT NOT NULL,
    `refresh_token` TEXT NOT NULL,
    `token_fingerprint` VARCHAR(64) NOT NULL,
    `scopes` VARCHAR(255) NOT NULL,
    `expires_at` DATETIME NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `user_id` (`user_id`),
    CONSTRAINT `fk_oauth_tokens_users` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE `oauth_tokens`;
-- +goose StatementEnd