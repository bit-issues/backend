package feed

import (
	"maps"
	"slices"
	"strconv"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
)

// maxFeedWindow bounds offset+limit so a single feed request cannot force the
// history query to load an unbounded window.
const maxFeedWindow = 5000

// deriveChanges compares a newer payload against its next-older snapshot and
// returns the keys that differ, in sorted-key order so renders stay
// deterministic. The vocabulary is the payload itself: writers decide what is
// tracked by writing it.
func deriveChanges(newer, older map[string]*string) []FieldChange {
	changes := make([]FieldChange, 0, len(newer))

	for _, field := range slices.Sorted(maps.Keys(newer)) {
		newValue := newer[field]
		oldValue := older[field]
		if !optionalStringsEqual(oldValue, newValue) {
			changes = append(changes, FieldChange{
				Field:    field,
				OldValue: oldValue,
				NewValue: newValue,
			})
		}
	}

	return changes
}

func optionalStringsEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// resolveChangeValues renders raw payload values for display: assignee ids
// become user names (falling back to "#id" placeholders for unknown users),
// everything else passes through unchanged. Nil pairs stay nil.
func resolveChangeValues(changes [][]FieldChange, usersMap map[int64]users.User) [][]FieldChange {
	if changes == nil {
		return nil
	}

	resolved := make([][]FieldChange, len(changes))
	for i := range changes {
		if changes[i] == nil {
			continue
		}
		resolved[i] = make([]FieldChange, len(changes[i]))
		for j := range changes[i] {
			change := changes[i][j]
			change.OldValue = resolveChangeValue(change.Field, change.OldValue, usersMap)
			change.NewValue = resolveChangeValue(change.Field, change.NewValue, usersMap)
			resolved[i][j] = change
		}
	}

	return resolved
}

// resolveChangeValue renders one payload value: the assignee field carries a
// user id, rendered as the user's name, a "#id" placeholder when the user is
// unknown, or the raw value when it does not parse as an id; all other fields
// pass through unchanged.
func resolveChangeValue(field string, value *string, usersMap map[int64]users.User) *string {
	if value == nil {
		return nil
	}

	if field == tasks.FieldAssignee {
		id, err := strconv.ParseInt(*value, 10, 64)
		if err == nil {
			if user, ok := usersMap[id]; ok {
				name := user.Name
				return &name
			}

			fallback := "#" + *value
			return &fallback
		}
	}

	resolved := *value
	return &resolved
}

// deriveHistoryChanges computes each entry's change pairs against its
// next-older snapshot; entries are newest-first, so that snapshot is the
// following element. Created entries carry no pairs; attachment pairs come
// from the payload keys and the action.
func deriveHistoryChanges(entries []history.Entry) [][]FieldChange {
	changes := make([][]FieldChange, len(entries))

	for i := range entries {
		entry := &entries[i]

		switch entry.Action {
		case history.ActionCreated:
			// Creation seeds the snapshot chain; it renders without pairs.
		case history.ActionUpdated:
			if i+1 < len(entries) {
				changes[i] = deriveChanges(entry.Payload, entries[i+1].Payload)
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
func attachmentFeedChanges(entry *history.Entry) []FieldChange {
	name := entry.Payload[attachments.FieldAttachment]
	size, hasSize := entry.Payload[attachments.FieldAttachmentSize]

	var changes []FieldChange

	if entry.Action == history.ActionAttachmentRemoved {
		changes = append(changes, FieldChange{
			Field:    attachments.FieldAttachment,
			OldValue: name,
			NewValue: nil,
		})
		if hasSize {
			changes = append(changes, FieldChange{
				Field:    attachments.FieldAttachmentSize,
				OldValue: size,
				NewValue: nil,
			})
		}

		return changes
	}

	changes = append(changes, FieldChange{
		Field:    attachments.FieldAttachment,
		OldValue: nil,
		NewValue: name,
	})
	if hasSize {
		changes = append(changes, FieldChange{
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
