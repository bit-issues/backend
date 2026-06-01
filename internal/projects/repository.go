package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
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
func (r *Repository) Create(ctx context.Context, input ProjectInput, slug string, tags []string) (*Project, error) {
	model := newProjectModel(input, slug)

	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrNameAlreadyUsed
		}
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	if len(tags) > 0 {
		if err := r.SetProjectTags(ctx, slug, tags); err != nil {
			return nil, fmt.Errorf("failed to set project tags: %w", err)
		}
	}

	return model.toDomain(tags), nil
}

// GetBySlug retrieves a project by its unique slug identifier.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Project, error) {
	var model projectModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", slug).Limit(1).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by slug: %w", err)
	}

	tags, err := r.getProjectTags(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get project tags: %w", err)
	}

	return model.toDomain(tags), nil
}

// List retrieves a paginated list of projects, ordered by name ascending.
func (r *Repository) List(ctx context.Context, pagination *Pagination, filter *ProjectFilter) ([]Project, int, error) {
	models := make([]projectModel, 0)

	query := r.db.NewSelect().Model(&models).OrderBy("name", bun.OrderAsc)

	if filter != nil && len(filter.Tags) > 0 {
		query = query.Where("p.id IN (?)",
			r.db.NewSelect().
				Column("pt.project_id").
				Model((*projectTagModel)(nil)).
				Where("pt.tag_name IN (?)", bun.List(filter.Tags)).
				Group("pt.project_id").
				Having("COUNT(DISTINCT pt.tag_name) = ?", len(filter.Tags)),
		)
	}

	query = pagination.Apply(query)
	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]Project, 0, len(models))
	if len(models) == 0 {
		return projects, total, nil
	}

	projectIDs := make([]string, len(models))
	for i, m := range models {
		projectIDs[i] = m.ID
	}

	tagMap, err := r.batchGetProjectTags(ctx, projectIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to batch get project tags: %w", err)
	}

	for _, m := range models {
		projects = append(projects, *m.toDomain(tagMap[m.ID]))
	}

	return projects, total, nil
}

// Update modifies an existing project with the provided update data.
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

	result, execErr := query.Exec(ctx)
	if execErr != nil {
		if db.IsUniqueViolation(execErr) {
			return ErrNameAlreadyUsed
		}
		return fmt.Errorf("failed to update project: %w", execErr)
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

// SetProjectTags replaces all tags for a project with the given names.
func (r *Repository) SetProjectTags(ctx context.Context, projectID string, tagNames []string) error {
	if _, err := r.db.NewDelete().
		Model((*projectTagModel)(nil)).
		Where("project_id = ?", projectID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete existing project tags: %w", err)
	}

	if len(tagNames) == 0 {
		return nil
	}

	junctions := make([]projectTagModel, len(tagNames))
	for i, name := range tagNames {
		junctions[i] = projectTagModel{
			BaseModel: schema.BaseModel{},
			ProjectID: projectID,
			TagName:   name,
		}
	}

	if _, err := r.db.NewInsert().Model(&junctions).Exec(ctx); err != nil {
		return fmt.Errorf("failed to insert project tags: %w", err)
	}

	return nil
}

// Delete removes a project from the database by its slug.
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
func (r *Repository) Exists(ctx context.Context, slug string) (bool, error) {
	exists, err := r.db.NewSelect().Model((*projectModel)(nil)).
		Where("id = ?", slug).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return exists, nil
}

// getProjectTags retrieves all tag names for a single project.
func (r *Repository) getProjectTags(ctx context.Context, projectID string) ([]string, error) {
	var models []projectTagModel
	if err := r.db.NewSelect().Model(&models).
		Where("project_id = ?", projectID).
		Order("tag_name ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get project tags: %w", err)
	}

	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.TagName
	}
	return names, nil
}

// batchGetProjectTags retrieves tags for multiple projects at once.
func (r *Repository) batchGetProjectTags(ctx context.Context, projectIDs []string) (map[string][]string, error) {
	var models []projectTagModel
	if err := r.db.NewSelect().Model(&models).
		Where("project_id IN (?)", bun.List(projectIDs)).
		Order("tag_name ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to batch get project tags: %w", err)
	}

	tagMap := make(map[string][]string, len(projectIDs))
	for _, m := range models {
		tagMap[m.ProjectID] = append(tagMap[m.ProjectID], m.TagName)
	}
	return tagMap, nil
}
