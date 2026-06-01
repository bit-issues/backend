package tags

import (
	"context"
	"fmt"
	"time"

	"github.com/go-core-fx/bunfx"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

// Repository provides data access for tags.
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new Repository instance.
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// EnsureExists creates any tags that don't already exist in the database.
// Already-existing tags are silently ignored (INSERT IGNORE).
func (r *Repository) EnsureExists(ctx context.Context, names []string) error {
	now := time.Now()
	for _, name := range names {
		if _, err := r.db.NewInsert().
			Model(&tagModel{
				BaseModel: schema.BaseModel{},
				TimedModel: bunfx.TimedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name: name,
			}).
			Ignore().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to ensure tag %q: %w", name, err)
		}
	}
	return nil
}
