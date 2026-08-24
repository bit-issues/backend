package projects

import (
	"context"
	"fmt"

	"github.com/gosimple/slug"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Service implements the business logic for project management.
// It coordinates between the repository layer and validation rules.
type Service struct {
	projects *Repository
}

// NewService creates a new Service instance with the given dependencies.
func NewService(repo *Repository) *Service {
	return &Service{
		projects: repo,
	}
}

// Create creates a new project after validating the input.
// Validation includes:
//   - Name must be non-empty after trimming whitespace
//   - Repository URL must be in valid format (HTTPS)
//   - Project name must be unique (case-sensitive)
//
// Returns the created project or an error if validation fails.
func (s *Service) Create(ctx context.Context, input ProjectInput) (*Project, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Create the project
	return s.projects.Create(ctx, input, slug.Make(input.Name))
}

// GetBySlug retrieves a project by its slug.
// Returns ErrNotFound if the project does not exist.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Project, error) {
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrValidationFailed)
	}
	return s.projects.GetBySlug(ctx, slug)
}

// List retrieves a paginated list of projects.
// Projects are ordered by name ascending.
// Supports optional search filtering by project name or slug.
func (s *Service) List(ctx context.Context, pagination *Pagination, search string) ([]Project, int, error) {
	return s.projects.List(ctx, pagination, search)
}

// Update modifies an existing project with the provided update data.
// Validation includes:
//   - At least one field must be provided
//   - If name is provided, it must be non-empty and unique
//   - If repoURL is provided, it must be in valid format
//
// Returns the updated project or an error if validation fails.
func (s *Service) Update(ctx context.Context, slug string, update ProjectUpdate) (*Project, error) {
	// Validate update data
	if err := update.Validate(); err != nil {
		return nil, err
	}

	// Perform update
	if err := s.projects.Update(ctx, slug, update); err != nil {
		return nil, err
	}

	// Return updated project
	return s.projects.GetBySlug(ctx, slug)
}

// Delete removes a project by its slug.
// All associated tasks, comments, and attachments will be cascade-deleted
// due to foreign key constraints in the database.
// Returns ErrNotFound if the project does not exist.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if slug == "" {
		return fmt.Errorf("%w: slug is required", ErrValidationFailed)
	}
	return s.projects.Delete(ctx, slug)
}

// Exists checks whether a project with the given slug exists.
// Returns true if found, false otherwise.
func (s *Service) Exists(ctx context.Context, slug string) (bool, error) {
	if slug == "" {
		return false, fmt.Errorf("%w: slug is required", ErrValidationFailed)
	}
	return s.projects.Exists(ctx, slug)
}

// FindByRepoURL finds a project whose repository URL matches the
// given one. Matching is case-insensitive.
func (s *Service) FindByRepoURL(ctx context.Context, repoURL string) (*Project, error) {
	if repoURL == "" {
		return nil, fmt.Errorf("%w: repoURL is required", ErrValidationFailed)
	}
	return s.projects.FindByRepoURL(ctx, repoURL)
}
