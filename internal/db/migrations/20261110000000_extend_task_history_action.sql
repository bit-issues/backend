-- +goose Up
-- +goose StatementBegin
ALTER TABLE `task_history`
MODIFY COLUMN `action` ENUM(
        'created',
        'updated',
        'attachment_added',
        'attachment_removed'
    ) NOT NULL;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DELETE FROM `task_history`
WHERE `action` IN ('attachment_added', 'attachment_removed');
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE `task_history`
MODIFY COLUMN `action` ENUM('created', 'updated') NOT NULL;
-- +goose StatementEnd