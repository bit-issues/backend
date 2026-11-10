// Package usecases coordinates cross-module operations that must commit
// atomically: a task mutation together with its activity-history entry, or an
// attachment operation together with its feed event.
//
// This is the application layer that owns database transactions. Repositories
// of the participating modules expose *Within methods that join the
// use case's transaction; module services stay free of transaction concerns
// and keep their read paths.
//
// Integration:
//   - Drives tasks.Repository, history.Repository, attachments.Repository
//   - Validates through the tasks/attachments domain types
//   - Provides usecases.Service for HTTP handlers, webhooks, and the importer
package usecases
