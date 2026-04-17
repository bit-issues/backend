package users

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

func (r *Repository) Create(ctx context.Context, input UserInput, passwordHash string) (*User, error) {
	model := newUserModel(input, passwordHash)
	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrEmailAlreadyUsed
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &model.toDomain().User, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*UserWithPasswordHash, error) {
	var model userModel
	if err := r.db.NewSelect().Model(&model).Where("email = ?", email).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return model.toDomain(), nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*UserWithPasswordHash, error) {
	var model userModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return model.toDomain(), nil
}

func (r *Repository) List(
	ctx context.Context,
	status *Status,
	role *Role,
	pagination *Pagination,
) ([]User, int, error) {
	models := make([]userModel, 0)
	query := r.db.NewSelect().Model(&models).OrderExpr("id DESC")

	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if role != nil {
		query = query.Where("role = ?", *role)
	}

	query = pagination.Apply(query)

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]User, 0, len(models))
	for _, model := range models {
		users = append(users, model.toDomain().User)
	}

	return users, total, nil
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	result, err := r.db.NewUpdate().
		Model((*userModel)(nil)).
		Set("password_hash = ?", passwordHash).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user password hash: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, id int64, update UserUpdate) error {
	if update.IsEmpty() {
		return nil
	}

	query := r.db.NewUpdate().Model((*userModel)(nil)).Where("id = ?", id)

	if update.Status != nil {
		query = query.Set("status = ?", *update.Status)
	}
	if update.Role != nil {
		query = query.Set("role = ?", *update.Role)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
