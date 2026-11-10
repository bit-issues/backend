package history

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// Repository handles data access operations for history entries.
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new Repository instance with the given database connection.
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// RecordWithin appends an entry using the caller's transaction so a
// cross-module use case can commit the event together with its trigger.
func (r *Repository) RecordWithin(ctx context.Context, idb bun.IDB, entry Entry) error {
	model := newHistoryModel(entry.TaskID, entry.ActorID, entry.Action, entry.Changes, entry.CreatedAt)
	if _, err := idb.NewInsert().Model(model).Exec(ctx); err != nil {
		return fmt.Errorf("failed to insert history entry: %w", err)
	}

	return nil
}

// List retrieves history entries for a task newest-first.
func (r *Repository) List(
	ctx context.Context,
	taskID int64,
	limit, offset int,
) ([]Entry, int, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	models := make([]historyModel, 0)
	query := r.db.NewSelect().
		Model(&models).
		Where("task_id = ?", taskID).
		OrderExpr("created_at DESC").
		OrderExpr("id DESC").
		Limit(limit).
		Offset(offset)

	total, err := query.ScanAndCount(ctx, &models)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list history entries: %w", err)
	}

	entries := make([]Entry, 0, len(models))
	for i := range models {
		entries = append(entries, *models[i].toDomain())
	}

	return entries, total, nil
}
