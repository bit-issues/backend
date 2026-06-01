// Package tags provides tag management for the corporate issue tracker.
//
// Tags are short labels that can be attached to projects (and potentially
// other entities) for categorization and filtering. Each tag is uniquely
// identified by its name (slug), which serves as the primary key.
//
// The module follows clean architecture principles:
//   - Domain layer (domain.go): Tag entity and input validation
//   - Data layer (models.go, repository.go): Tags table persistence
//   - Service layer (service.go): Business logic
//
// Integration:
//   - Used by projects.Service to manage project tags
//   - Uses bun.DB for MySQL persistence
//   - Provides tags.Service for other modules
//
// Example usage:
//
//	app := fx.New(
//	    tags.Module(),
//	    fx.Invoke(func(svc *tags.Service) {
//	        svc.EnsureExists(ctx, "bug", "feature")
//	    }),
//	)
package tags
