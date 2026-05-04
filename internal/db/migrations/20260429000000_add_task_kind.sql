-- +goose Up
-- +goose StatementBegin
ALTER TABLE `tasks`
ADD COLUMN `kind` ENUM('Bug', 'Enhancement', 'Task', 'Proposal') NOT NULL DEFAULT 'Task'
AFTER `status`;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
ALTER TABLE `tasks` DROP COLUMN `kind`;
-- +goose StatementEnd