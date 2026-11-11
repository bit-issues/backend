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

// Entry is one append-only activity event about a subject: the writer's
// EntryBase input plus the timestamp history filled in.
type Entry struct {
	EntryBase

	ID        int64
	CreatedAt time.Time
}

// EntryBase is the input writers hand to Service.Record: they decide the
// subject type, the action and carry the actual written values as an opaque
// payload - this module neither knows nor interprets the keys. It carries no
// timestamp; history fills CreatedAt on the stored Entry itself.
type EntryBase struct {
	SubjectType string
	SubjectID   int64
	ActorID     int64
	Action      Action
	Payload     map[string]*string
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
