package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
)

// Repository is the concrete implementation of Repository using Bun ORM.
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new Repository instance with the given database connection.
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new project into the database.
// Returns the created project, or an error if creation fails.
// Expected errors:
//   - ErrNameAlreadyUsed if a project with the same name already exists
//   - Other database errors
func (r *Repository) Create(ctx context.Context, input ProjectInput, slug string) (*Project, error) {
	model := newProjectModel(input, slug)

	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrNameAlreadyUsed
		}
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return model.toDomain(), nil
}

// GetBySlug retrieves a project by its unique slug identifier.
// Returns ErrNotFound if the project does not exist.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Project, error) {
	var model projectModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", slug).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by slug: %w", err)
	}
	return model.toDomain(), nil
}

// List retrieves a paginated list of projects, ordered by name ascending.
// Returns the list of projects and no error if successful.
func (r *Repository) List(ctx context.Context, pagination *Pagination) ([]Project, int, error) {
	models := make([]projectModel, 0)

	query := r.db.NewSelect().Model(&models).OrderBy("name", bun.OrderAsc)
	query = pagination.Apply(query)
	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]Project, 0, len(models))
	for _, model := range models {
		projects = append(projects, *model.toDomain())
	}

	return projects, total, nil
}

// Update modifies an existing project with the provided update data.
// Only the fields specified in the update struct will be changed.
// Returns ErrNotFound if the project does not exist.
// Returns ErrNameAlreadyUsed if the new name conflicts with another project.
func (r *Repository) Update(ctx context.Context, slug string, update ProjectUpdate) error {
	if update.IsEmpty() {
		return nil
	}

	query := r.db.NewUpdate().Model((*projectModel)(nil)).Where("id = ?", slug)

	if update.Name != nil {
		query = query.Set("name = ?", *update.Name)
	}
	if update.RepoURL != nil {
		query = query.Set("repo_url = ?", *update.RepoURL)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return ErrNameAlreadyUsed
		}
		return fmt.Errorf("failed to update project: %w", err)
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

// Delete removes a project from the database by its slug.
// Due to foreign key constraints with ON DELETE CASCADE, all associated
// tasks, comments, and attachments will be automatically deleted.
// Returns ErrNotFound if the project does not exist.
func (r *Repository) Delete(ctx context.Context, slug string) error {
	result, err := r.db.NewDelete().Model((*projectModel)(nil)).Where("id = ?", slug).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
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

// Exists checks whether a project with the given slug exists.
// Returns true if found, false otherwise.
func (r *Repository) Exists(ctx context.Context, slug string) (bool, error) {
	exists, err := r.db.NewSelect().Model((*projectModel)(nil)).
		Where("id = ?", slug).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return exists, nil
}
