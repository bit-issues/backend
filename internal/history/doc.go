// Package history records and serves the append-only activity stream of the
// issue tracker: field-level task changes, attachment events, and the seed
// entries written when a task is created.
//
// The module owns the action vocabulary and the EntryBase (input) / Entry
// (output) pair, and the
// history table - a polymorphic, append-only stream keyed by
// (subject_type, subject_id). Subject types and field names are opaque
// strings here: writers decide what they record, readers give the strings
// meaning.
// Repositories own the transaction and invoke a writer-provided AuditFunc so
// event and trigger commit atomically.
//
// Integration:
//   - Provides history.Service for HTTP handlers (feed composition) and for
//     writing modules (Record)
//   - Provides history.AuditFunc for repository audit callbacks
//   - Depends on bun.DB for MySQL persistence
package history
