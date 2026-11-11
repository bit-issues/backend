// Package history records and serves the append-only activity stream of the
// issue tracker: field-level task changes, attachment events, and the seed
// entries written when a task is created.
//
// The module owns the event vocabulary (subject types, actions and field
// names), the entry builders, and the history table - a polymorphic,
// append-only stream keyed by (subject_type, subject_id). A writing module
// starts its own transaction, writes its data, then passes the transaction to
// a Service record method so event and trigger commit atomically.
//
// Integration:
//   - Provides history.Service for HTTP handlers (feed composition) and for
//     writing modules (RecordCreated/RecordUpdated/RecordAttachment*)
//   - Depends on bun.DB for MySQL persistence
package history
