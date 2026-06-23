package webauthn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, model *credentialModel) error {
	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		if db.IsUniqueViolation(err) {
			return ErrDuplicateCredential
		}
		return fmt.Errorf("failed to create credential: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Credential, error) {
	var model credentialModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("failed to get credential by id: %w", err)
	}
	return model.toDomain(), nil
}

func (r *Repository) GetByCredentialID(ctx context.Context, credentialID []byte) (*Credential, error) {
	var model credentialModel
	if err := r.db.NewSelect().
		Model(&model).
		Where("credential_id = ?", CredentialID(credentialID).String()).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("failed to get credential by credential_id: %w", err)
	}
	return model.toDomain(), nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID int64) ([]Credential, error) {
	models := make([]credentialModel, 0)
	if err := r.db.NewSelect().
		Model(&models).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get credentials by user id: %w", err)
	}

	creds := make([]Credential, 0, len(models))
	for _, m := range models {
		creds = append(creds, *m.toDomain())
	}
	return creds, nil
}

func (r *Repository) UpdateSignCount(ctx context.Context, id int64, signCount uint32) error {
	result, err := r.db.NewUpdate().
		Model((*credentialModel)(nil)).
		Set("sign_count = ?", signCount).
		Where("id = ?", id).
		Where("sign_count <= ?", signCount).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update sign count: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		exists, selErr := r.db.NewSelect().Model((*credentialModel)(nil)).Where("id = ?", id).Exists(ctx)
		if selErr != nil {
			return fmt.Errorf("failed to check credential existence: %w", selErr)
		}
		if !exists {
			return ErrCredentialNotFound
		}
		return nil
	}
	return nil
}

func (r *Repository) UpdateName(ctx context.Context, id int64, userID int64, name string) error {
	result, err := r.db.NewUpdate().
		Model((*credentialModel)(nil)).
		Set("name = ?", name).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update credential name: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64, userID int64) error {
	result, err := r.db.NewDelete().
		Model((*credentialModel)(nil)).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrCredentialNotFound
	}
	return nil
}
