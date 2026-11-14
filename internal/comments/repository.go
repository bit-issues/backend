package comments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

// Repository handles data access operations for comments.
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new Repository instance with the given database connection.
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new comment.
func (r *Repository) Create(ctx context.Context, input CommentInput) (*Comment, error) {
	model := newCommentModel(input)

	if _, err := r.db.NewInsert().Model(model).Returning("*").Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	return model.toDomain(), nil
}

// Import creates a comment with explicit timestamps for import.
func (r *Repository) Import(
	ctx context.Context,
	input Comment,
) (*Comment, error) {
	model := newCommentImport(input)

	if _, err := r.db.NewInsert().Model(model).Returning("*").Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to insert comment for import: %w", err)
	}

	return model.toDomain(), nil
}

// GetByID retrieves a comment by its ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Comment, error) {
	var model commentModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get comment by ID: %w", err)
	}
	return model.toDomain(), nil
}

// List retrieves comments with filtering, sorting, and pagination.
// A nil pagination returns all matching comments.
func (r *Repository) List(
	ctx context.Context,
	filter CommentFilter,
	sort SortOrder,
	pagination *Pagination,
) ([]Comment, int, error) {
	models := make([]commentModel, 0)
	query := r.db.NewSelect().Model(&models)

	query = filter.apply(query)
	query = sort.apply(query)

	if pagination != nil {
		query = pagination.Apply(query)
	}

	total, err := query.ScanAndCount(ctx, &models)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list comments: %w", err)
	}

	comments := make([]Comment, 0, len(models))
	for i := range models {
		comments = append(comments, *models[i].toDomain())
	}

	return comments, total, nil
}

// Update modifies an existing comment with the provided content.
func (r *Repository) Update(ctx context.Context, id int64, content string) error {
	if _, err := r.db.NewUpdate().
		Model((*commentModel)(nil)).
		Set("content = ?", content).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// Delete soft-deletes a comment by setting the deleted_at timestamp.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.NewDelete().
		Model((*commentModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}
