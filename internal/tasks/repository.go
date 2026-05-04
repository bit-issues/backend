package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
)

// Repository handles data access operations for tasks.
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new Repository instance with the given database connection.
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new task with an auto-generated per-project number.
// The number generation happens within a transaction to ensure uniqueness.
func (r *Repository) Create(ctx context.Context, input TaskInput) (*Task, error) {
	var task *Task

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get the next task number for this project
		var maxNumber int
		err := tx.NewSelect().
			Model((*taskModel)(nil)).
			Column("number").
			Where("project_slug = ?", input.ProjectSlug).
			WhereAllWithDeleted().
			OrderExpr("number DESC").
			Limit(1).
			Scan(ctx, &maxNumber)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to get max task number: %w", err)
		}

		nextNumber := maxNumber + 1
		model := newTaskModel(input, nextNumber)

		if _, insErr := tx.NewInsert().Model(model).Exec(ctx); insErr != nil {
			if db.IsUniqueViolation(insErr) {
				return fmt.Errorf(
					"%w: task number %d already exists for project %s",
					ErrValidationFailed,
					nextNumber,
					input.ProjectSlug,
				)
			}
			return fmt.Errorf("failed to insert task: %w", insErr)
		}

		task = model.toDomain()
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	ok, err := r.db.NewSelect().Model((*taskModel)(nil)).Where("id = ?", id).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %w", err)
	}
	return ok, nil
}

// GetByID retrieves a task by its global database ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Task, error) {
	var model taskModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task by ID: %w", err)
	}
	return model.toDomain(), nil
}

// List retrieves a paginated list of tasks with optional filtering and sorting.
// The filter parameter controls which tasks are returned. Empty fields mean no filter.
// The sort parameter is a field name, optionally prefixed with "-" for descending order.
func (r *Repository) List(
	ctx context.Context,
	filter TaskFilter,
	sort string,
	pagination *Pagination,
) ([]Task, int, error) {
	models := make([]taskModel, 0)
	query := r.db.NewSelect().Model(&models)

	// Apply filters
	query = filter.apply(query)

	// Apply sorting
	switch sort {
	case "", "-created_at", "created_at":
		if sort == "-created_at" || sort == "" {
			query = query.OrderBy("created_at", bun.OrderDesc)
		} else {
			query = query.OrderBy("created_at", bun.OrderAsc)
		}
	case "-priority", "priority":
		if sort == "-priority" {
			query = query.OrderBy("priority", bun.OrderDesc)
		} else {
			query = query.OrderBy("priority", bun.OrderAsc)
		}
	case "-due_date", "due_date":
		if sort == "-due_date" {
			query = query.OrderBy("due_date", bun.OrderDesc)
		} else {
			query = query.OrderBy("due_date", bun.OrderAsc)
		}
	case "-status", "status":
		if sort == "-status" {
			query = query.OrderBy("status", bun.OrderDesc)
		} else {
			query = query.OrderBy("status", bun.OrderAsc)
		}
	}

	// Apply pagination
	query = pagination.Apply(query)

	total, err := query.ScanAndCount(ctx, &models)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]Task, 0, len(models))
	for _, model := range models {
		tasks = append(tasks, *model.toDomain())
	}

	return tasks, total, nil
}

// Update modifies an existing task with the provided update data.
// Only non-nil fields in the TaskUpdate struct will be changed.
func (r *Repository) Update(ctx context.Context, id int64, update TaskUpdate) error {
	if update.IsEmpty() {
		return nil
	}

	query := r.db.NewUpdate().Model((*taskModel)(nil)).Where("id = ?", id)

	// Build the SET clause dynamically
	if update.Title != nil {
		query = query.Set("title = ?", *update.Title)
	}
	if update.Description != nil {
		query = query.Set("description = ?", *update.Description)
	}
	if update.Priority != nil {
		query = query.Set("priority = ?", *update.Priority)
	}
	if update.Status != nil {
		query = query.Set("status = ?", *update.Status)
	}
	if update.Kind != nil {
		query = query.Set("kind = ?", *update.Kind)
	}
	if update.AssigneeID != nil {
		if *update.AssigneeID == 0 {
			query = query.Set("assignee_id = NULL")
		} else {
			query = query.Set("assignee_id = ?", *update.AssigneeID)
		}
	}
	if update.DueDate != nil {
		if *update.DueDate == "" {
			query = query.Set("due_date = NULL")
		} else {
			query = query.Set("due_date = ?", *update.DueDate)
		}
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
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

// Delete soft-deletes a task.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.NewDelete().Model((*taskModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
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
