package jwt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	model := newRefreshTokenModel(userID, tokenHash, expiresAt)
	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *Repository) FindByHash(ctx context.Context, tokenHash string) (*Token, error) {
	model := new(refreshTokenModel)
	if err := r.db.NewSelect().Model(model).
		Where("token_hash = ?", tokenHash).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // special case
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	return model.toDomain(), nil
}

func (r *Repository) RevokeByHash(ctx context.Context, tokenHash string) error {
	result, err := r.db.NewUpdate().
		Model((*refreshTokenModel)(nil)).
		Set("revoked = ?", true).
		Where("token_hash = ?", tokenHash).
		Where("revoked = ?", false).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrRefreshTokenRevoked
	}
	return nil
}

func (r *Repository) RevokeAllForUser(ctx context.Context, userID int64) error {
	if _, err := r.db.NewUpdate().
		Model((*refreshTokenModel)(nil)).
		Set("revoked = ?", true).
		Where("user_id = ?", userID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens for user: %w", err)
	}
	return nil
}
