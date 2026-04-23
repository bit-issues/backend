-- +goose Up
-- +goose StatementBegin
ALTER TABLE `users`
ADD COLUMN `name` VARCHAR(255) NULL;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE `users`
SET `name` = SUBSTRING_INDEX(`email`, '@', 1)
WHERE `name` IS NULL
    OR `name` = '';
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE `users`
MODIFY COLUMN `name` VARCHAR(255) NOT NULL;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_users_status_lower_name_prefix ON users (`status`, `name`);
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_users_status_lower_name_prefix ON users;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE `users` DROP COLUMN `name`;
-- +goose StatementEnd