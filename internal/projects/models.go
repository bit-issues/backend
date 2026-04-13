package projects

import (
	"time"

	"github.com/go-core-fx/bunfx"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

// projectModel is the database representation of a project.
// This struct is used for Bun ORM operations and maps to the `projects` table.
type projectModel struct {
	bun.BaseModel `bun:"table:projects,alias:p"`
	bunfx.TimedModel

	ID      string `bun:"id,pk"` // Primary key
	Name    string `bun:"name,notnull,unique"`
	RepoURL string `bun:"repo_url,notnull"`
}

// newProjectModel creates a new projectModel from a ProjectInput.
// It automatically sets the ID from the name and timestamps.
func newProjectModel(input ProjectInput, slug string) *projectModel {
	now := time.Now()
	return &projectModel{
		BaseModel: schema.BaseModel{},
		TimedModel: bunfx.TimedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},

		ID:      slug,
		Name:    input.Name,
		RepoURL: input.RepoURL,
	}
}

// toDomain converts the database model to a domain Project entity.
// Returns nil if the model is nil.
func (m *projectModel) toDomain() *Project {
	if m == nil {
		return nil
	}
	return &Project{
		ID:        m.ID,
		Name:      m.Name,
		RepoURL:   m.RepoURL,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
