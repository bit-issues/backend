// Package tasks provides task management functionality for the corporate issue tracker.
//
// This module implements the core task lifecycle management including:
//   - Task creation with per-project auto-incrementing numbers
//   - Status and priority management (matching BitBucket enums)
//   - Assignment and due date tracking
//   - Soft delete support for audit trails
//   - Filtering and sorting for dashboards
//
// The module follows clean architecture principles with clear separation between:
//   - Domain layer (domain.go): Business entities and validation
//   - Data layer (models.go, repository.go): Persistence and queries
//   - Service layer (service.go): Business logic and coordination
//
// Integration:
//   - Depends on projects.Service for project existence validation
//   - Uses bun.DB for MySQL persistence
//   - Provides tasks.Service for HTTP handlers
//
// Example usage:
//
//	// In your FX application setup:
//	app := fx.New(
//	    tasks.Module(),
//	    fx.Invoke(func(service *tasks.Service) {
//	        // Use the service
//	    }),
//	)
//
// The module is production-ready and follows the established patterns from
// the internal/example, internal/users, and internal/projects modules.
package tasks
