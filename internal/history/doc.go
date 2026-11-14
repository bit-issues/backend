// Package history records and serves the append-only activity stream of the
// issue tracker: field-level task changes, attachment events, and the seed
// entries written when a task is created.
//
// The module owns the action vocabulary and the EntryBase (input) / Entry
// (output) pair, and the history table - a polymorphic, append-only stream
// keyed by (subject_type, subject_id). Subject types, field names and the
// payload values are opaque here: writers save their domain data first, then
// record best-effort with the actual written values; readers turn payloads
// into human-meaningful changes.
//
// Integration:
//   - Provides history.Service for HTTP handlers (feed composition) and for
//     writing modules (Record)
//   - Depends on bun.DB for MySQL persistence
package history
