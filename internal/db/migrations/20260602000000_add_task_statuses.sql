-- +goose Up
-- +goose StatementBegin
ALTER TABLE `tasks`
MODIFY COLUMN `status` ENUM(
        'New',
        'Open',
        'In Progress',
        'Resolved',
        'Closed',
        'Reopened',
        'Invalid',
        'Duplicate',
        'Wontfix',
        'On Hold'
    ) NOT NULL DEFAULT 'New';
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
UPDATE `tasks`
SET `status` = 'Closed'
WHERE `status` IN ('Invalid', 'Duplicate', 'Wontfix');
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE `tasks`
MODIFY COLUMN `status` ENUM(
        'New',
        'Open',
        'In Progress',
        'Resolved',
        'Closed',
        'Reopened'
    ) NOT NULL DEFAULT 'New';
-- +goose StatementEnd