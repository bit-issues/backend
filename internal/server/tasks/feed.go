package tasks

import (
	"slices"
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
)

type feedEntryKind string

// maxFeedWindow bounds offset+limit so a single feed request cannot force the
// comment and history queries to load an unbounded window.
const maxFeedWindow = 5000

const (
	feedEntryKindComment feedEntryKind = "comment"
	feedEntryKindHistory feedEntryKind = "history"
)

type feedEntry struct {
	Kind      feedEntryKind
	ID        int64
	CreatedAt time.Time
	Comment   *comments.Comment
	History   *history.Entry
	Changes   []tasks.FieldChange
}

func newCommentFeedEntry(comment *comments.Comment) feedEntry {
	return feedEntry{
		Kind:      feedEntryKindComment,
		ID:        comment.ID,
		CreatedAt: comment.CreatedAt,
		Comment:   comment,
		History:   nil,
		Changes:   nil,
	}
}

func newHistoryFeedEntry(entry *history.Entry, changes []tasks.FieldChange) feedEntry {
	return feedEntry{
		Kind:      feedEntryKindHistory,
		ID:        entry.ID,
		CreatedAt: entry.CreatedAt,
		Comment:   nil,
		History:   entry,
		Changes:   changes,
	}
}

func mergeFeedEntries(
	commentList []comments.Comment,
	historyList []history.Entry,
	historyChanges [][]tasks.FieldChange,
) []feedEntry {
	entries := make([]feedEntry, 0, len(commentList)+len(historyList))

	for i := range commentList {
		entries = append(entries, newCommentFeedEntry(&commentList[i]))
	}
	for i := range historyList {
		entries = append(entries, newHistoryFeedEntry(&historyList[i], historyChanges[i]))
	}

	slices.SortFunc(entries, compareFeedEntries)

	return entries
}

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

func compareFeedEntries(a, b feedEntry) int {
	if a.CreatedAt.After(b.CreatedAt) {
		return -1
	}
	if a.CreatedAt.Before(b.CreatedAt) {
		return 1
	}

	if a.Kind != b.Kind {
		if a.Kind == feedEntryKindComment {
			return -1
		}
		return 1
	}

	if a.ID > b.ID {
		return -1
	}
	if a.ID < b.ID {
		return 1
	}

	return 0
}

// feedWindow returns the merged-feed query window (offset+limit), reporting
// false when it would exceed maxFeedWindow.
func feedWindow(offset, limit int) (int, bool) {
	if offset > maxFeedWindow-limit {
		return 0, false
	}

	return offset + limit, true
}

func pageFeedEntries(entries []feedEntry, offset, limit int) []feedEntry {
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	if offset >= len(entries) {
		return make([]feedEntry, 0)
	}

	end := min(offset+limit, len(entries))
	return entries[offset:end]
}
