package tasks

import (
	"context"

	"go.uber.org/zap"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Service implements the business logic for task management.
// Mutations that must record history atomically live in the use-case layer;
// the service owns validation-free reads and soft deletion.
type Service struct {
	tasks  *Repository
	logger *zap.Logger
}

// NewService creates a new Service instance with the given dependencies.
func NewService(repo *Repository, logger *zap.Logger) *Service {
	return &Service{
		tasks:  repo,
		logger: logger,
	}
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

// Delete soft-deletes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.tasks.Delete(ctx, id)
}
