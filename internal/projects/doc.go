// Package projects provides project management functionality for the
// Corporate Task Tracker. Projects serve as containers for tasks and
// are linked to external BitBucket repositories.
//
// The projects module follows a clean architecture pattern with clear
// separation between domain logic, data access, and HTTP presentation.
//
// Domain Layer:
//   - Project: Core business entity
//   - ProjectInput: Data for creating projects
//   - ProjectUpdate: Data for updating projects
//
// Repository Layer:
//   - Handles all database operations
//   - Uses Bun ORM for type-safe queries
//
// Service Layer:
//   - Implements business rules and validation
//   - Validates repository URLs
//   - Ensures name uniqueness
//
// HTTP Layer:
//   - RESTful API endpoints
//   - Admin-only write operations
//   - Authenticated read operations
//
// Usage:
//
//	app := fx.New(
//	    projects.Module(),
//	    // other modules...
//	)
package projects
