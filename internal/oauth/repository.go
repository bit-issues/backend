package oauth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

// Repository persists the per-user OAuth token row. Access and refresh tokens
// are encrypted at rest via enc before being written, and decrypted on read.
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, userID int64, token EncryptedToken) error {
	model := newTokenModel(userID, token)

	if _, err := r.db.NewInsert().Model(model).On("DUPLICATE KEY UPDATE").Exec(ctx); err != nil {
		return fmt.Errorf("failed to upsert token: %w", err)
	}

	return nil
}

// Update persists a refreshed token only if the credential loaded before the
// refresh still exists and matches currentFingerprint. It returns false when
// no row matched, which happens if the token was deleted (or replaced) while
// the refresh request was in flight.
func (r *Repository) Update(
	ctx context.Context,
	userID int64,
	currentFingerprint string,
	token EncryptedToken,
) (bool, error) {
	res, err := r.db.NewUpdate().
		Model((*tokenModel)(nil)).
		Set("access_token = ?", token.AccessToken).
		Set("refresh_token = ?", token.RefreshToken).
		Set("token_fingerprint = ?", token.Fingerprint).
		Set("scopes = ?", token.Scopes).
		Set("expires_at = ?", token.ExpiresAt).
		Set("updated_at = ?", token.UpdatedAt).
		Where("user_id = ?", userID).
		Where("token_fingerprint = ?", currentFingerprint).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to update token: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read affected rows: %w", err)
	}

	return n > 0, nil
}

func (r *Repository) Get(ctx context.Context, userID int64) (*EncryptedToken, error) {
	var model tokenModel
	if err := r.db.NewSelect().Model(&model).
		Where("user_id = ?", userID).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	token := model.toDomain()

	return token, nil
}

func (r *Repository) Delete(ctx context.Context, userID int64) error {
	if _, err := r.db.NewDelete().Model((*tokenModel)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

// fingerprint returns a stable SHA-256 hex digest of the plaintext refresh
// token. Because AES-GCM seals with a random nonce, the stored ciphertext is
// not suitable for optimistic-concurrency checks; the fingerprint is.
func fingerprint(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
