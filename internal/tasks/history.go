package tasks

import (
	"strconv"
	"time"
)

type HistoryAction string

const (
	HistoryActionCreated           HistoryAction = "created"
	HistoryActionUpdated           HistoryAction = "updated"
	HistoryActionAttachmentAdded   HistoryAction = "attachment_added"
	HistoryActionAttachmentRemoved HistoryAction = "attachment_removed"
)

const (
	HistoryFieldTitle       = "title"
	HistoryFieldDescription = "description"
	HistoryFieldPriority    = "priority"
	HistoryFieldStatus      = "status"
	HistoryFieldKind        = "kind"
	HistoryFieldAssignee    = "assignee"
	HistoryFieldDueDate     = "due_date"
	HistoryFieldAttachment  = "attachment"
)

type FieldChange struct {
	Field    string  `json:"field"`
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

type TaskHistoryEntry struct {
	ID        int64
	TaskID    int64
	ActorID   int64
	Action    HistoryAction
	Changes   []FieldChange
	CreatedAt time.Time
}

func diffTask(current *Task, update TaskUpdate) []FieldChange {
	changes := make([]FieldChange, 0)

	if update.Title != nil && *update.Title != current.Title {
		changes = append(changes, newStringChange(HistoryFieldTitle, current.Title, *update.Title))
	}
	if update.Description != nil && *update.Description != current.Description {
		changes = append(changes, newStringChange(HistoryFieldDescription, current.Description, *update.Description))
	}
	if update.Priority != nil && *update.Priority != current.Priority {
		changes = append(
			changes,
			newStringChange(HistoryFieldPriority, string(current.Priority), string(*update.Priority)),
		)
	}
	if update.Status != nil && *update.Status != current.Status {
		changes = append(changes, newStringChange(HistoryFieldStatus, string(current.Status), string(*update.Status)))
	}
	if update.Kind != nil && *update.Kind != current.Kind {
		changes = append(changes, newStringChange(HistoryFieldKind, string(current.Kind), string(*update.Kind)))
	}
	if update.AssigneeID != nil {
		currentAssignee := formatAssignee(current.AssigneeID)
		newAssignee := formatAssignee(update.AssigneeID)
		if !optionalStringsEqual(currentAssignee, newAssignee) {
			changes = append(changes, newOptionalChange(HistoryFieldAssignee, currentAssignee, newAssignee))
		}
	}
	if update.DueDate != nil {
		currentDueDate := current.DueDate
		var newDueDate *string
		if *update.DueDate != "" {
			newDueDate = update.DueDate
		}
		if !optionalStringsEqual(currentDueDate, newDueDate) {
			changes = append(changes, newOptionalChange(HistoryFieldDueDate, currentDueDate, newDueDate))
		}
	}

	return changes
}

func newStringChange(field, oldValue, newValue string) FieldChange {
	oldCopy := oldValue
	newCopy := newValue
	return FieldChange{
		Field:    field,
		OldValue: &oldCopy,
		NewValue: &newCopy,
	}
}

func newOptionalChange(field string, oldValue, newValue *string) FieldChange {
	return FieldChange{
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
	}
}

// attachmentChange builds the field change carried by an attachment history
// entry. The file name lands in NewValue for additions and in OldValue for
// removals, mirroring the before/after state of other field changes.
func attachmentChange(action HistoryAction, fileName string) FieldChange {
	name := fileName
	if action == HistoryActionAttachmentRemoved {
		return newOptionalChange(HistoryFieldAttachment, &name, nil)
	}

	return newOptionalChange(HistoryFieldAttachment, nil, &name)
}

func formatAssignee(id *int64) *string {
	if id == nil || *id == 0 {
		return nil
	}

	value := strconv.FormatInt(*id, 10)
	return &value
}

func optionalStringsEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
