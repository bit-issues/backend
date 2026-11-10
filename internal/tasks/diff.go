package tasks

import (
	"strconv"

	"github.com/bit-issues/backend/internal/history"
)

// diffTask computes the field-level changes between the current task state
// and an update. An empty result means the update is a no-op.
func diffTask(current *Task, update TaskUpdate) []history.FieldChange {
	changes := make([]history.FieldChange, 0)

	if update.Title != nil && *update.Title != current.Title {
		changes = append(changes, history.NewStringChange(history.FieldTitle, current.Title, *update.Title))
	}
	if update.Description != nil && *update.Description != current.Description {
		changes = append(
			changes,
			history.NewStringChange(history.FieldDescription, current.Description, *update.Description),
		)
	}
	if update.Priority != nil && *update.Priority != current.Priority {
		changes = append(
			changes,
			history.NewStringChange(history.FieldPriority, string(current.Priority), string(*update.Priority)),
		)
	}
	if update.Status != nil && *update.Status != current.Status {
		changes = append(
			changes,
			history.NewStringChange(history.FieldStatus, string(current.Status), string(*update.Status)),
		)
	}
	if update.Kind != nil && *update.Kind != current.Kind {
		changes = append(
			changes,
			history.NewStringChange(history.FieldKind, string(current.Kind), string(*update.Kind)),
		)
	}
	if update.AssigneeID != nil {
		currentAssignee := formatAssignee(current.AssigneeID)
		newAssignee := formatAssignee(update.AssigneeID)
		if !optionalStringsEqual(currentAssignee, newAssignee) {
			changes = append(changes, history.NewOptionalChange(history.FieldAssignee, currentAssignee, newAssignee))
		}
	}
	if update.DueDate != nil {
		currentDueDate := current.DueDate
		var newDueDate *string
		if *update.DueDate != "" {
			newDueDate = update.DueDate
		}
		if !optionalStringsEqual(currentDueDate, newDueDate) {
			changes = append(changes, history.NewOptionalChange(history.FieldDueDate, currentDueDate, newDueDate))
		}
	}

	return changes
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
