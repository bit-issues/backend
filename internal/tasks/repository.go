package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/db"
	"github.com/bit-issues/backend/internal/history"
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

// CreateWithin inserts a new task with an auto-generated per-project number
// inside the caller's transaction, so a use case can commit the task together
// with its history entry atomically.
func (r *Repository) CreateWithin(ctx context.Context, idb bun.IDB, input TaskInput) (*Task, error) {
	// Get the next task number for this project
	var maxNumber int
	err := idb.NewSelect().
		Model((*taskModel)(nil)).
		Column("number").
		Where("project_slug = ?", input.ProjectSlug).
		WhereAllWithDeleted().
		OrderExpr("number DESC").
		Limit(1).
		Scan(ctx, &maxNumber)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get max task number: %w", err)
	}

	nextNumber := maxNumber + 1
	model := newTaskModel(input, nextNumber)

	if _, insErr := idb.NewInsert().Model(model).Exec(ctx); insErr != nil {
		if db.IsUniqueViolation(insErr) {
			return nil, fmt.Errorf(
				"%w: task number %d already exists for project %s",
				ErrValidationFailed,
				nextNumber,
				input.ProjectSlug,
			)
		}
		return nil, fmt.Errorf("failed to insert task: %w", insErr)
	}

	return model.toDomain(), nil
}

// ImportWithin creates a task with explicit number, timestamps, and kind
// inside the caller's transaction. It bypasses auto-number generation and
// allows setting custom CreatedAt/UpdatedAt.
func (r *Repository) ImportWithin(
	ctx context.Context,
	idb bun.IDB,
	input Task,
) (*Task, error) {
	model := newTaskImport(input)

	if _, insErr := idb.NewInsert().Model(model).Exec(ctx); insErr != nil {
		if db.IsUniqueViolation(insErr) {
			return nil, fmt.Errorf(
				"%w: task number %d already exists for project %s",
				ErrValidationFailed,
				input.Number,
				input.ProjectSlug,
			)
		}
		return nil, fmt.Errorf("failed to insert task for import: %w", insErr)
	}

	return model.toDomain(), nil
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

// GetByProjectAndNumber retrieves a task by its project slug and number.
// Returns ErrNotFound if no task matches.
func (r *Repository) GetByProjectAndNumber(ctx context.Context, projectSlug string, number int) (*Task, error) {
	var model taskModel
	if err := r.db.NewSelect().Model(&model).
		Where("project_slug = ?", projectSlug).
		Where("number = ?", number).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task by project and number: %w", err)
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

	query = query.OrderBy("id", bun.OrderDesc)

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

// UpdateWithin modifies an existing task with the provided update data
// inside the caller's transaction and returns the applied field changes, so a
// use case can record them in the same transaction. Only non-nil fields in
// the TaskUpdate struct are changed.
func (r *Repository) UpdateWithin(
	ctx context.Context,
	idb bun.IDB,
	id int64,
	update TaskUpdate,
) (*Task, []history.FieldChange, error) {
	if update.IsEmpty() {
		current, getErr := r.GetByID(ctx, id)
		if getErr != nil {
			return nil, nil, getErr
		}
		return current, nil, nil
	}

	current, fetchErr := getTaskForUpdate(ctx, idb, id)
	if fetchErr != nil {
		return nil, nil, fetchErr
	}

	changes := diffTask(current, update)
	if len(changes) == 0 {
		return current, nil, nil
	}

	rows, updateErr := executeTaskUpdate(ctx, idb, id, update)
	if updateErr != nil {
		return nil, nil, updateErr
	}
	if rows == 0 {
		return nil, nil, ErrNotFound
	}

	after, reloadErr := getTaskByIDWithDB(ctx, idb, id)
	if reloadErr != nil {
		return nil, nil, reloadErr
	}

	return after, changes, nil
}

func getTaskForUpdate(ctx context.Context, idb bun.IDB, id int64) (*Task, error) {
	var model taskModel
	if err := idb.NewSelect().
		Model(&model).
		Where("id = ?", id).
		For("UPDATE").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task for update: %w", err)
	}

	return model.toDomain(), nil
}

func getTaskByIDWithDB(ctx context.Context, idb bun.IDB, id int64) (*Task, error) {
	var model taskModel
	if err := idb.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to reload updated task: %w", err)
	}

	return model.toDomain(), nil
}

func executeTaskUpdate(
	ctx context.Context,
	idb bun.IDB,
	id int64,
	update TaskUpdate,
) (int64, error) {
	query := buildTaskUpdateQuery(idb, update).Where("id = ?", id)
	result, err := query.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rows, nil
}

func buildTaskUpdateQuery(idb bun.IDB, update TaskUpdate) *bun.UpdateQuery {
	query := idb.NewUpdate().Model((*taskModel)(nil))

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

	return query
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
