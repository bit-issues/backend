package projects

import (
	"context"
	"fmt"

	"github.com/bit-issues/backend/internal/tags"
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
	tags     *tags.Service
}

// NewService creates a new Service instance with the given repository and tags service.
func NewService(repo *Repository, tagsSvc *tags.Service) *Service {
	return &Service{projects: repo, tags: tagsSvc}
}

// Create creates a new project after validating the input.
// Validation includes:
//   - Name must be non-empty after trimming whitespace
//   - Repository URL must be in valid format (HTTPS)
//   - Tags must be non-empty and within max length
//   - Project name must be unique (case-sensitive)
//
// Returns the created project or an error if validation fails.
func (s *Service) Create(ctx context.Context, input ProjectInput) (*Project, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	tags := input.Tags
	if len(tags) > 0 {
		var err error
		if tags, err = s.tags.EnsureExists(ctx, tags); err != nil {
			return nil, fmt.Errorf("failed to ensure tags: %w", err)
		}
	}

	return s.projects.Create(ctx, input, slug.Make(input.Name), tags)
}

// GetBySlug retrieves a project by its slug.
// Returns ErrNotFound if the project does not exist.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Project, error) {
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrValidationFailed)
	}
	return s.projects.GetBySlug(ctx, slug)
}

// List retrieves a paginated list of projects, optionally filtered.
// Projects are ordered by name ascending.
func (s *Service) List(ctx context.Context, pagination *Pagination, filter *ProjectFilter) ([]Project, int, error) {
	return s.projects.List(ctx, pagination, filter)
}

// Update modifies an existing project with the provided update data.
// Validation includes:
//   - At least one field must be provided
//   - If name is provided, it must be non-empty and unique
//   - If repoURL is provided, it must be in valid format
//   - If tags are provided, they must be non-empty and within max length
//
// Returns the updated project or an error if validation fails.
func (s *Service) Update(ctx context.Context, projectSlug string, update ProjectUpdate) (*Project, error) {
	// Validate update data
	if err := update.Validate(); err != nil {
		return nil, err
	}

	// Update project fields
	if err := s.projects.Update(ctx, projectSlug, update); err != nil {
		return nil, err
	}

	// Update tags if provided
	if update.Tags != nil {
		tags := *update.Tags
		if len(tags) > 0 {
			var err error
			if tags, err = s.tags.EnsureExists(ctx, tags); err != nil {
				return nil, fmt.Errorf("failed to ensure tags: %w", err)
			}
		}
		if err := s.projects.SetProjectTags(ctx, projectSlug, tags); err != nil {
			return nil, fmt.Errorf("failed to set project tags: %w", err)
		}
	}

	return s.projects.GetBySlug(ctx, projectSlug)
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
