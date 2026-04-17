package projects

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/db"
)

// Project represents the core business entity for a project.
// A project is a container for tasks and is linked to a BitBucket repository.
type Project struct {
	ID        string    // Primary key
	Name      string    // Unique project name
	RepoURL   string    // BitBucket repository URL
	CreatedAt time.Time // Creation timestamp
	UpdatedAt time.Time // Last update timestamp
}

// ProjectInput represents the data required to create a new project.
// All fields are required and must be validated before creation.
type ProjectInput struct {
	Name    string // Unique project name, required
	RepoURL string // BitBucket repository URL, required
}

// ProjectUpdate represents the data that can be updated for a project.
// All fields are optional (pointers) to support partial updates.
type ProjectUpdate struct {
	Name    *string // Optional new name, must be unique if provided
	RepoURL *string // Optional new repository URL, must be valid if provided
}

// Validate checks if the input data is valid for creating a new project.
func (i ProjectInput) Validate() error {
	// Trim and validate name
	name := strings.TrimSpace(i.Name)
	if name == "" {
		return fmt.Errorf("%w: project name is required", ErrValidationFailed)
	}

	// Validate repository URL
	repoURL := strings.TrimSpace(i.RepoURL)
	if err := validateRepoURL(repoURL); err != nil {
		return err
	}

	return nil
}

// IsEmpty returns true if no update fields are set.
// This prevents unnecessary database operations when no data is provided.
func (u ProjectUpdate) IsEmpty() bool {
	return u.Name == nil && u.RepoURL == nil
}

func (u ProjectUpdate) Validate() error {
	if u.Name != nil {
		// Trim and validate name
		name := strings.TrimSpace(*u.Name)
		if name == "" {
			return fmt.Errorf("%w: project name is required", ErrValidationFailed)
		}
	}

	if u.RepoURL != nil {
		// Validate repository URL
		repoURL := strings.TrimSpace(*u.RepoURL)
		if err := validateRepoURL(repoURL); err != nil {
			return err
		}
	}

	return nil
}

// validateRepoURL validates that the repository URL is in a valid format.
// Accepts HTTPS URLs (https://bitbucket.org/...).
func validateRepoURL(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("%w: repository URL is required", ErrValidationFailed)
	}

	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("%w: failed to parse repository URL: %w", ErrValidationFailed, err)
	}

	// Accept HTTPS scheme
	if u.Scheme != "https" {
		return fmt.Errorf("%w: repository URL must be in HTTPS format", ErrValidationFailed)
	}

	// Basic validation: must have host
	if u.Host == "" {
		return fmt.Errorf("%w: repository URL must have a host", ErrValidationFailed)
	}

	return nil
}

type pagination struct {
}

// DefaultLimit implements [db.DefaultPagination].
func (p *pagination) DefaultLimit() int {
	return DefaultLimit
}

// MaxLimit implements [db.DefaultPagination].
func (p *pagination) MaxLimit() int {
	return MaxLimit
}

var _ db.DefaultPagination = (*pagination)(nil)

type Pagination = db.Pagination[*pagination]

func NewPagination(limit, offset int) *Pagination {
	return db.NewPagination[*pagination](limit, offset)
}
