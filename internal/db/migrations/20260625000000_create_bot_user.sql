-- +goose Up
-- +goose StatementBegin
INSERT INTO users (
        email,
        name,
        password_hash,
        role,
        status,
        created_at,
        updated_at
    )
VALUES (
        'bot@bitissues.local',
        'BitBucket Bot',
        'x',
        'admin',
        'active',
        NOW(),
        NOW()
    ) ON DUPLICATE KEY
UPDATE name =
VALUES(name),
    role =
VALUES(role),
    status =
VALUES(status);
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DELETE FROM users
WHERE email = 'bot@bitissues.local';
-- +goose StatementEnd