// Package feed reads the task activity feed: it turns history entries into
// field-level change pairs - a generic payload-key diff for task snapshots,
// action-directed extraction for attachment events - over the vocabulary of
// the modules that record them (tasks, attachments) and the action vocabulary
// of history.
//
// The module owns the read side of the activity stream - window bounding,
// read-side change derivation and the boundary-row context fetch. Writers
// record their events themselves (tasks, attachments); this module only
// consumes.
//
// Integration:
//   - Provides feed.Service for HTTP handlers (feed endpoint)
//   - Depends on history.Service for reads, users.Service for actor and
//     assignee enrichment, and a named logger for degradation logging
package feed
