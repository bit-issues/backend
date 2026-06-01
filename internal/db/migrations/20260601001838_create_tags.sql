-- +goose Up
-- +goose StatementBegin
CREATE TABLE `tags` (
    `name` VARCHAR(255) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE `project_tags` (
    `project_id` VARCHAR(255) NOT NULL,
    `tag_name` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`project_id`, `tag_name`),
    KEY `idx_project_tags_tag_name` (`tag_name`),
    CONSTRAINT `fk_project_tags_project` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_project_tags_tag` FOREIGN KEY (`tag_name`) REFERENCES `tags` (`name`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE `project_tags`;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE `tags`;
-- +goose StatementEnd
