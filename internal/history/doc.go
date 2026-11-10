// Package history records and serves the append-only activity stream of the
// issue tracker: field-level task changes, attachment events, and the seed
// entries written when a task is created.
//
// The module owns the event vocabulary (actions and field names), the entry
// constructors, and the task_history table. Writes happen either through the
// use-case layer, which passes its own transaction to RecordWithin, or through
// reads via the Service.
//
// Integration:
//   - Provides history.Service for HTTP handlers (feed composition)
//   - Provides history.Repository for the use-case layer (cross-module tx)
//   - Depends on bun.DB for MySQL persistence
package history
