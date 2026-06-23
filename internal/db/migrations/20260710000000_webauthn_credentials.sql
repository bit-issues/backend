-- +goose Up
-- +goose StatementBegin
CREATE TABLE webauthn_credentials (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    credential_id VARCHAR(255) NOT NULL,
    public_key BLOB NOT NULL,
    attestation_type VARCHAR(64) NOT NULL DEFAULT '',
    transport JSON,
    aaguid CHAR(36) NOT NULL DEFAULT '',
    flags TINYINT UNSIGNED NOT NULL DEFAULT 0,
    sign_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_wc_credential_id (credential_id),
    KEY idx_wc_user_id (user_id),
    CONSTRAINT fk_wc_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS webauthn_credentials;
-- +goose StatementEnd