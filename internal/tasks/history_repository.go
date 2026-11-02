package tasks

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func (r *Repository) insertHistory(ctx context.Context, idb bun.IDB, history *historyModel) error {
	if _, err := idb.NewInsert().Model(history).Exec(ctx); err != nil {
		return fmt.Errorf("failed to insert task history: %w", err)
	}
	return nil
}

// ListHistory retrieves task history entries newest-first.
func (r *Repository) ListHistory(
	ctx context.Context,
	taskID int64,
	limit, offset int,
) ([]TaskHistoryEntry, int, error) {
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
		return nil, 0, fmt.Errorf("failed to list task history: %w", err)
	}

	entries := make([]TaskHistoryEntry, 0, len(models))
	for i := range models {
		entries = append(entries, *models[i].toDomain())
	}

	return entries, total, nil
}
