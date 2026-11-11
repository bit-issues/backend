package attachments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/history"
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

// Confirm marks a pending attachment uploaded and invokes audit with the row
// inside the same transaction. The status flip is conditional, so a
// re-confirm of an already uploaded attachment is a no-op without audit.
func (r *Repository) Confirm(ctx context.Context, id int64, audit history.AuditFunc[*Attachment]) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, updErr := tx.NewUpdate().
			Model((*attachmentModel)(nil)).
			Set("status = ?", StatusUploaded).
			Where("id = ?", id).
			Where("status = ?", StatusPending).
			Exec(ctx)
		if updErr != nil {
			return fmt.Errorf("failed to update attachment status: %w", updErr)
		}

		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("failed to get affected rows: %w", rowsErr)
		}
		if rows == 0 {
			return nil
		}

		var model attachmentModel
		if selErr := tx.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx); selErr != nil {
			return fmt.Errorf("failed to get attachment for audit: %w", selErr)
		}

		return audit(ctx, tx, model.toDomain())
	})
	if err != nil {
		return fmt.Errorf("failed to confirm attachment: %w", err)
	}

	return nil
}

// Delete removes an attachment row and invokes audit with the row inside the
// same transaction.
func (r *Repository) Delete(ctx context.Context, id int64, audit history.AuditFunc[*Attachment]) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var model attachmentModel
		if selErr := tx.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx); selErr != nil {
			if errors.Is(selErr, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("failed to get attachment for audit: %w", selErr)
		}

		if _, delErr := tx.NewDelete().Model((*attachmentModel)(nil)).Where("id = ?", id).Exec(ctx); delErr != nil {
			return fmt.Errorf("failed to delete attachment row: %w", delErr)
		}

		return audit(ctx, tx, model.toDomain())
	})
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}
