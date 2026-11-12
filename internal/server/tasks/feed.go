package tasks

import (
	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
)

// maxFeedWindow bounds offset+limit so a single feed request cannot force the
// history query to load an unbounded window.
const maxFeedWindow = 5000

// deriveHistoryChanges computes each entry's change pairs against its
// next-older snapshot; entries are newest-first, so that snapshot is the
// following element. Created entries carry no pairs; attachment pairs come
// from the payload keys and the action.
func deriveHistoryChanges(entries []history.Entry) [][]tasks.FieldChange {
	changes := make([][]tasks.FieldChange, len(entries))

	for i := range entries {
		entry := &entries[i]

		switch entry.Action {
		case history.ActionCreated:
			// Creation seeds the snapshot chain; it renders without pairs.
		case history.ActionUpdated:
			if i+1 < len(entries) {
				changes[i] = tasks.DeriveChanges(entry.Payload, entries[i+1].Payload)
			}
		case history.ActionAttachmentAdded, history.ActionAttachmentRemoved:
			changes[i] = attachmentFeedChanges(entry)
		}
	}

	return changes
}

// attachmentFeedChanges renders the attachment name pair, plus a size pair
// when the payload carries one (rows recorded before size_bytes existed omit
// it). Removed events put the values on the old side.
func attachmentFeedChanges(entry *history.Entry) []tasks.FieldChange {
	name := entry.Payload[attachments.FieldAttachment]
	size, hasSize := entry.Payload[attachments.FieldAttachmentSize]

	var changes []tasks.FieldChange

	if entry.Action == history.ActionAttachmentRemoved {
		changes = append(changes, tasks.FieldChange{
			Field:    attachments.FieldAttachment,
			OldValue: name,
			NewValue: nil,
		})
		if hasSize {
			changes = append(changes, tasks.FieldChange{
				Field:    attachments.FieldAttachmentSize,
				OldValue: size,
				NewValue: nil,
			})
		}

		return changes
	}

	changes = append(changes, tasks.FieldChange{
		Field:    attachments.FieldAttachment,
		OldValue: nil,
		NewValue: name,
	})
	if hasSize {
		changes = append(changes, tasks.FieldChange{
			Field:    attachments.FieldAttachmentSize,
			OldValue: nil,
			NewValue: size,
		})
	}

	return changes
}

// feedWindow returns the requested history query window (offset+limit),
// reporting false when it would exceed maxFeedWindow.
func feedWindow(offset, limit int) (int, bool) {
	if offset > maxFeedWindow-limit {
		return 0, false
	}

	return offset + limit, true
}
