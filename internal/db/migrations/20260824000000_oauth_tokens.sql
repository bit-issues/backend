-- +goose Up
-- +goose StatementBegin
CREATE TABLE `oauth_tokens` (
    `singleton_id` TINYINT UNSIGNED NOT NULL,
    `access_token` VARCHAR(1024) NOT NULL,
    `refresh_token` VARCHAR(1024) NOT NULL,
    `scope` VARCHAR(255) NOT NULL,
    `expires_at` DATETIME NOT NULL,
    `connected_by_user_id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`singleton_id`),
    CONSTRAINT `fk_oauth_tokens_user` FOREIGN KEY (`connected_by_user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
-- +goose StatementEnd
---
-- +goose StatementBegin
CREATE TABLE `oauth_states` (
    `state_hash` CHAR(64) NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `redirect_uri` VARCHAR(2048) NOT NULL,
    `expires_at` DATETIME NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`state_hash`),
    KEY `idx_oauth_states_user_id` (`user_id`),
    KEY `idx_oauth_states_expires_at` (`expires_at`),
    CONSTRAINT `fk_oauth_states_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE `oauth_states`;
DROP TABLE `oauth_tokens`;
-- +goose StatementEnd
