package tasks

import (
	"context"
	"fmt"

	"github.com/bit-issues/backend/internal/projects"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

// GetByID retrieves a task by its global ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Task, error) {
	return s.tasks.GetByID(ctx, id)
}

// List retrieves tasks with filtering, sorting, and pagination.
// Returns the list of tasks and the total count matching the filter.
func (s *Service) List(
	ctx context.Context,
	filter TaskFilter,
	sort string,
	pagination *Pagination,
) ([]Task, int64, error) {
	var tasks []Task
	var total int64

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		tasks, err = s.tasks.List(ctx, filter, sort, pagination)
		return err
	})
	g.Go(func() error {
		var err error
		total, err = s.tasks.Count(ctx, filter)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// func (s *Service) ListByUser(ctx context.Context, userID int64, sort string, limit, offset int) ([]Task, int64, error) {

// }

// Update modifies an existing task with the provided data.
// Returns the updated task or an error if not found or validation fails.
func (s *Service) Update(ctx context.Context, id int64, update TaskUpdate) (*Task, error) {
	if update.IsEmpty() {
		return nil, fmt.Errorf("%w: no update data provided", ErrValidationFailed)
	}

	if err := update.Validate(); err != nil {
		return nil, err
	}

	// Perform update
	if err := s.tasks.Update(ctx, id, update); err != nil {
		return nil, err
	}

	// Return the updated task
	return s.tasks.GetByID(ctx, id)
}

// Delete soft-deletes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.tasks.Delete(ctx, id)
}
