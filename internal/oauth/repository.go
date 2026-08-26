package oauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

// Repository persists the singleton OAuth token row and CSRF states.
type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, userID int64, token *Token) error {
	model := newTokenModel(userID, token)
	if _, err := r.db.NewInsert().Model(model).On("DUPLICATE KEY UPDATE").Exec(ctx); err != nil {
		return fmt.Errorf("failed to upsert token: %w", err)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, userID int64) (*Token, error) {
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
	return model.toDomain(), nil
}

func (r *Repository) Delete(ctx context.Context, userID int64) error {
	if _, err := r.db.NewDelete().Model((*tokenModel)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}
