package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/bit-issues/backend/internal/projects"
	"go.uber.org/zap"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Service implements the business logic for task management.
type Service struct {
	tasks    *Repository
	projects *projects.Service
	logger   *zap.Logger
}

// NewService creates a new Service instance with the given dependencies.
func NewService(repo *Repository, projects *projects.Service, logger *zap.Logger) *Service {
	return &Service{
		tasks:    repo,
		projects: projects,
		logger:   logger,
	}
}

// Create validates input and creates a new task with auto-generated number.
func (s *Service) Create(ctx context.Context, input TaskInput) (*Task, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Verify project exists
	exists, err := s.projects.Exists(ctx, input.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to verify project: %w", err)
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	// Create task (repository handles number generation in transaction)
	return s.tasks.Create(ctx, input)
}

// Import creates a task with explicit number, timestamps, and kind for import.
// Bypasses auto-number generation and allows preserving original metadata.
func (s *Service) Import(
	ctx context.Context,
	input Task,
) (*Task, error) {
	// Create task with import-specific data
	return s.tasks.Import(ctx, input)
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

// Update modifies an existing task with the provided data.
// Returns the updated task or an error if not found or validation fails.
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
		return nil, err
	}

	if actorID <= 0 {
		return nil, fmt.Errorf("%w: actor_id must be positive", ErrValidationFailed)
	}

	return s.tasks.Update(ctx, id, update, actorID)
}

// ListHistory retrieves task history entries newest-first.
func (s *Service) ListHistory(
	ctx context.Context,
	taskID int64,
	limit, offset int,
) ([]TaskHistoryEntry, int, error) {
	return s.tasks.ListHistory(ctx, taskID, limit, offset)
}

// RecordAttachmentAdded appends an attachment-added entry to the task history
// so a confirmed upload shows up in the activity feed.
func (s *Service) RecordAttachmentAdded(ctx context.Context, taskID, actorID int64, fileName string) error {
	return s.recordAttachmentEvent(ctx, taskID, actorID, HistoryActionAttachmentAdded, fileName)
}

// RecordAttachmentRemoved appends an attachment-removed entry to the task
// history so a deletion shows up in the activity feed.
func (s *Service) RecordAttachmentRemoved(ctx context.Context, taskID, actorID int64, fileName string) error {
	return s.recordAttachmentEvent(ctx, taskID, actorID, HistoryActionAttachmentRemoved, fileName)
}

func (s *Service) recordAttachmentEvent(
	ctx context.Context,
	taskID, actorID int64,
	action HistoryAction,
	fileName string,
) error {
	return s.tasks.recordHistory(ctx, newHistoryModel(
		taskID,
		actorID,
		action,
		[]FieldChange{attachmentChange(action, fileName)},
		time.Now(),
	))
}

// Delete soft-deletes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.tasks.Delete(ctx, id)
}
