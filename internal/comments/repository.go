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

// List retrieves all comments for a specific task.
func (r *Repository) List(ctx context.Context, taskID int64) ([]Comment, error) {
	models := make([]commentModel, 0)

	query := r.db.NewSelect().Model(&models).Where("task_id = ?", taskID)

	if err := query.OrderExpr("created_at ASC").Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to list comments by task: %w", err)
	}

	comments := make([]Comment, 0, len(models))
	for _, model := range models {
		comments = append(comments, *model.toDomain())
	}

	return comments, nil
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
