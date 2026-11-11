package history

import "time"

// Action is the kind of event recorded in the activity history.
type Action string

const (
	// ActionCreated marks the creation of a subject (a task).
	ActionCreated Action = "created"
	// ActionUpdated marks a field-level change to a subject.
	ActionUpdated Action = "updated"
	// ActionAttachmentAdded marks a confirmed attachment upload.
	ActionAttachmentAdded Action = "attachment_added"
	// ActionAttachmentRemoved marks a deleted attachment.
	ActionAttachmentRemoved Action = "attachment_removed"
)

// FieldChange is a single change carried by a history entry. Field is an
// opaque string to this module - field names and their semantics belong to
// the writers and readers of the stream.
type FieldChange struct {
	Field    string  `json:"field"`
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

// Entry is one append-only activity event about a subject: the writer's
// EntryBase input plus the timestamp history filled in.
type Entry struct {
	EntryBase

	ID        int64
	CreatedAt time.Time
}

// NewStringChange builds a field change whose old and new values are strings.
func NewStringChange(field, oldValue, newValue string) FieldChange {
	oldCopy := oldValue
	newCopy := newValue
	return FieldChange{
		Field:    field,
		OldValue: &oldCopy,
		NewValue: &newCopy,
	}
}

// NewOptionalChange builds a field change whose values may be absent.
func NewOptionalChange(field string, oldValue, newValue *string) FieldChange {
	return FieldChange{
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
	}
}

// EntryBase is the input writers hand to Service.Record: they decide the
// subject type, the action and the change payload. It carries no timestamp -
// history fills CreatedAt on the stored Entry itself.
type EntryBase struct {
	SubjectType string
	SubjectID   int64
	ActorID     int64
	Action      Action
	Changes     []FieldChange
}

// entryFromBase converts an input base into the stored entry, stamped with
// the provided timestamp.
func entryFromBase(base EntryBase, createdAt time.Time) Entry {
	return Entry{
		ID:        0,
		EntryBase: base,
		CreatedAt: createdAt,
	}
}
