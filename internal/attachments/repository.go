package attachments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Import(ctx context.Context, model *attachmentModel) (*Attachment, error) {
	if _, err := r.db.NewInsert().Model(model).Returning("*").Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to import attachment: %w", err)
	}

	return model.toDomain(), nil
}

func (r *Repository) Create(ctx context.Context, input AttachmentInput, storageKey string) (*Attachment, error) {
	model := newAttachmentModel(input, storageKey)

	if _, err := r.db.NewInsert().Model(model).Returning("*").Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	return model.toDomain(), nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Attachment, error) {
	var model attachmentModel
	if err := r.db.NewSelect().
		Model(&model).
		Where("id = ?", id).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get attachment by id: %w", err)
	}

	return model.toDomain(), nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID int64) ([]Attachment, error) {
	models := make([]attachmentModel, 0)
	if err := r.db.NewSelect().
		Model(&models).
		Where("task_id = ?", taskID).
		Where("status = ?", StatusUploaded).
		OrderExpr("uploaded_at ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}

	result := make([]Attachment, 0, len(models))
	for _, item := range models {
		result = append(result, *item.toDomain())
	}

	return result, nil
}

// Confirm marks a pending attachment uploaded and reports whether the status
// actually flipped; a re-confirm of an already uploaded attachment is a
// no-op without an event.
func (r *Repository) Confirm(ctx context.Context, id int64) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*attachmentModel)(nil)).
		Set("status = ?", StatusUploaded).
		Where("id = ?", id).
		Where("status = ?", StatusPending).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to update attachment status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rows > 0, nil
}

// Delete removes an attachment row.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	if _, err := r.db.NewDelete().Model((*attachmentModel)(nil)).Where("id = ?", id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}
