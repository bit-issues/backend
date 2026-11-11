package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Service implements the business logic for task management. Mutations run in
// a transaction that also appends the matching activity-history event.
type Service struct {
	tasks       *Repository
	projectsSvc *projects.Service
	historySvc  *history.Service
	db          *bun.DB
	logger      *zap.Logger
}

// NewService creates a new Service instance with the given dependencies.
func NewService(
	repo *Repository,
	db *bun.DB,
	projectsSvc *projects.Service,
	historySvc *history.Service,
	logger *zap.Logger,
) *Service {
	return &Service{
		tasks:       repo,
		projectsSvc: projectsSvc,
		historySvc:  historySvc,
		db:          db,
		logger:      logger,
	}
}

// Create validates input, creates a new task with an auto-generated number
// and records its creation in the history, atomically.
func (s *Service) Create(ctx context.Context, input TaskInput) (*Task, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task input: %w", err)
	}

	exists, err := s.projectsSvc.Exists(ctx, input.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to verify project: %w", err)
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	var task *Task
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		created, createErr := s.tasks.CreateWithin(ctx, tx, input)
		if createErr != nil {
			return fmt.Errorf("transaction: %w", createErr)
		}
		task = created

		if recordErr := s.historySvc.RecordCreated(
			ctx, tx, created.ID, created.AuthorID, created.CreatedAt,
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// Import creates a task with explicit number, timestamps, and kind for
// import, recording its creation in the history, atomically.
// Bypasses auto-number generation and allows preserving original metadata.
func (s *Service) Import(
	ctx context.Context,
	input Task,
) (*Task, error) {
	var task *Task
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		imported, importErr := s.tasks.ImportWithin(ctx, tx, input)
		if importErr != nil {
			return fmt.Errorf("transaction: %w", importErr)
		}
		task = imported

		if recordErr := s.historySvc.RecordCreated(
			ctx, tx, imported.ID, imported.AuthorID, imported.CreatedAt,
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to import task: %w", err)
	}

	return task, nil
}

// Exists checks if a task with the given ID exists.
func (s *Service) Exists(ctx context.Context, id int64) (bool, error) {
	return s.tasks.Exists(ctx, id)
}

// GetByID retrieves a task by its global ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Task, error) {
	return s.tasks.GetByID(ctx, id)
}

// GetByProjectAndNumber retrieves a task by its project slug and number.
func (s *Service) GetByProjectAndNumber(ctx context.Context, projectSlug string, number int) (*Task, error) {
	return s.tasks.GetByProjectAndNumber(ctx, projectSlug, number)
}

// List retrieves tasks with filtering, sorting, and pagination.
// Returns the list of tasks and the total count matching the filter.
func (s *Service) List(
	ctx context.Context,
	filter TaskFilter,
	sort string,
	pagination *Pagination,
) ([]Task, int, error) {
	return s.tasks.List(ctx, filter, sort, pagination)
}

// Update modifies an existing task with the provided data and records the
// applied changes in the history, atomically. Returns the updated task or an
// error if not found or validation fails.
func (s *Service) Update(
	ctx context.Context,
	id int64,
	update TaskUpdate,
	actorID int64,
) (*Task, error) {
	if update.IsEmpty() {
		return nil, fmt.Errorf("%w: no update data provided", ErrValidationFailed)
	}

	if err := update.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task update: %w", err)
	}

	if actorID <= 0 {
		return nil, fmt.Errorf("%w: actor_id must be positive", ErrValidationFailed)
	}

	var updated *Task
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		task, changes, updateErr := s.tasks.UpdateWithin(ctx, tx, id, update)
		if updateErr != nil {
			return fmt.Errorf("transaction: %w", updateErr)
		}
		updated = task

		if len(changes) == 0 {
			return nil
		}

		if recordErr := s.historySvc.RecordUpdated(
			ctx, tx, id, actorID, changes, time.Now(),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return updated, nil
}

// Delete soft-deletes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.tasks.Delete(ctx, id)
}
