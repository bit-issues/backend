-- +goose Up
-- +goose StatementBegin
CREATE FULLTEXT INDEX idx_tasks_search ON tasks (title, description);
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_tasks_search ON tasks;
-- +goose StatementEnd