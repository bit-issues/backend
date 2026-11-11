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

// Field names carried by history entry changes.
const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldPriority    = "priority"
	FieldStatus      = "status"
	FieldKind        = "kind"
	FieldAssignee    = "assignee"
	FieldDueDate     = "due_date"
	FieldAttachment  = "attachment"
)

// SubjectType identifies what a history entry is about.
type SubjectType string

const (
	// SubjectTask marks entries about a task.
	SubjectTask SubjectType = "task"
)

// FieldChange is a single field-level change carried by a history entry.
type FieldChange struct {
	Field    string  `json:"field"`
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

// Entry is one append-only activity event about a subject.
type Entry struct {
	ID          int64
	SubjectType SubjectType
	SubjectID   int64
	ActorID     int64
	Action      Action
	Changes     []FieldChange
	CreatedAt   time.Time
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

// newCreatedEntry builds the entry recording a task creation.
func newCreatedEntry(taskID, actorID int64, createdAt time.Time) Entry {
	return Entry{
		ID:          0,
		SubjectType: SubjectTask,
		SubjectID:   taskID,
		ActorID:     actorID,
		Action:      ActionCreated,
		Changes:     nil,
		CreatedAt:   createdAt,
	}
}

// newUpdatedEntry builds the entry recording applied field changes.
func newUpdatedEntry(taskID, actorID int64, changes []FieldChange, createdAt time.Time) Entry {
	return Entry{
		ID:          0,
		SubjectType: SubjectTask,
		SubjectID:   taskID,
		ActorID:     actorID,
		Action:      ActionUpdated,
		Changes:     changes,
		CreatedAt:   createdAt,
	}
}

// newAttachmentAddedEntry builds the entry recording a confirmed upload; the
// file name lands in NewValue, mirroring before/after state semantics.
func newAttachmentAddedEntry(taskID, actorID int64, fileName string, createdAt time.Time) Entry {
	return Entry{
		ID:          0,
		SubjectType: SubjectTask,
		SubjectID:   taskID,
		ActorID:     actorID,
		Action:      ActionAttachmentAdded,
		Changes:     []FieldChange{newAttachmentChange(ActionAttachmentAdded, fileName)},
		CreatedAt:   createdAt,
	}
}

// newAttachmentRemovedEntry builds the entry recording a deletion; the file
// name lands in OldValue, mirroring before/after state semantics.
func newAttachmentRemovedEntry(taskID, actorID int64, fileName string, createdAt time.Time) Entry {
	return Entry{
		ID:          0,
		SubjectType: SubjectTask,
		SubjectID:   taskID,
		ActorID:     actorID,
		Action:      ActionAttachmentRemoved,
		Changes:     []FieldChange{newAttachmentChange(ActionAttachmentRemoved, fileName)},
		CreatedAt:   createdAt,
	}
}

func newAttachmentChange(action Action, fileName string) FieldChange {
	name := fileName
	if action == ActionAttachmentRemoved {
		return NewOptionalChange(FieldAttachment, &name, nil)
	}

	return NewOptionalChange(FieldAttachment, nil, &name)
}
