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

// SaveTokens upserts the singleton OAuth token row.
func (r *Repository) SaveTokens(ctx context.Context, token *Token) error {
	model := newTokenModel(token)

	if _, err := r.db.NewInsert().Model(model).
		On("DUPLICATE KEY UPDATE").
		Set("access_token = VALUES(access_token)").
		Set("refresh_token = VALUES(refresh_token)").
		Set("scope = VALUES(scope)").
		Set("expires_at = VALUES(expires_at)").
		Set("connected_by_user_id = VALUES(connected_by_user_id)").
		Set("updated_at = VALUES(updated_at)").
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to save oauth tokens: %w", err)
	}

	return nil
}

// GetToken returns the singleton OAuth token row.
func (r *Repository) GetToken(ctx context.Context) (*Token, error) {
	var model tokenModel

	if err := r.db.NewSelect().
		Model(&model).
		Where("singleton_id = ?", SingletonID).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOAuthNotConnected
		}
		return nil, fmt.Errorf("failed to get oauth token: %w", err)
	}

	return model.toDomain(), nil
}

// DeleteTokens removes the singleton OAuth token row. Deleting an absent row
// is not an error, so disconnect stays idempotent.
func (r *Repository) DeleteTokens(ctx context.Context) error {
	if _, err := r.db.NewDelete().
		Model((*tokenModel)(nil)).
		Where("singleton_id = ?", SingletonID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete oauth tokens: %w", err)
	}

	return nil
}

// CreateState stores a CSRF state row keyed by its SHA-256 hash.
func (r *Repository) CreateState(ctx context.Context, state *State) error {
	model := newStateModel(state)

	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		return fmt.Errorf("failed to create oauth state: %w", err)
	}

	return nil
}

// GetState returns a CSRF state row by its SHA-256 hash.
func (r *Repository) GetState(ctx context.Context, stateHash string) (*State, error) {
	var model stateModel

	if err := r.db.NewSelect().
		Model(&model).
		Where("state_hash = ?", stateHash).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStateNotFound
		}
		return nil, fmt.Errorf("failed to get oauth state: %w", err)
	}

	return model.toDomain(), nil
}

// DeleteState removes a CSRF state row by its SHA-256 hash and reports
// whether exactly one row was deleted. The state hash is the primary key, so
// a result of zero rows means the state was already consumed or never
// existed. Callers must treat zero rows as a rejection, not a no-op.
func (r *Repository) DeleteState(ctx context.Context, stateHash string) (bool, error) {
	res, err := r.db.NewDelete().
		Model((*stateModel)(nil)).
		Where("state_hash = ?", stateHash).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to delete oauth state: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read deleted oauth state count: %w", err)
	}

	return affected == 1, nil
}
