package tasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/history"
	"github.com/uptrace/bun"
)

// SubjectTypeTask is the activity-history subject type written for task
// events; history stores subject types as opaque strings and this module
// decides the value.
const SubjectTypeTask = "task"

// Field names emitted by task diffs. These are the task-activity vocabulary:
// this module gives them meaning (diffing, assignee ids), the feed DTO and
// frontend render them - history only stores them as opaque strings.
const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldPriority    = "priority"
	FieldStatus      = "status"
	FieldKind        = "kind"
	FieldAssignee    = "assignee"
	FieldDueDate     = "due_date"
)

// diffTask computes the field-level changes between the current task state
// and an update. An empty result means the update is a no-op.
func diffTask(current *Task, update TaskUpdate) []history.FieldChange {
	changes := make([]history.FieldChange, 0)

	if update.Title != nil && *update.Title != current.Title {
		changes = append(changes, history.NewStringChange(FieldTitle, current.Title, *update.Title))
	}
	if update.Description != nil && *update.Description != current.Description {
		changes = append(
			changes,
			history.NewStringChange(FieldDescription, current.Description, *update.Description),
		)
	}
	if update.Priority != nil && *update.Priority != current.Priority {
		changes = append(
			changes,
			history.NewStringChange(FieldPriority, string(current.Priority), string(*update.Priority)),
		)
	}
	if update.Status != nil && *update.Status != current.Status {
		changes = append(
			changes,
			history.NewStringChange(FieldStatus, string(current.Status), string(*update.Status)),
		)
	}
	if update.Kind != nil && *update.Kind != current.Kind {
		changes = append(
			changes,
			history.NewStringChange(FieldKind, string(current.Kind), string(*update.Kind)),
		)
	}
	if update.AssigneeID != nil {
		currentAssignee := formatAssignee(current.AssigneeID)
		newAssignee := formatAssignee(update.AssigneeID)
		if !optionalStringsEqual(currentAssignee, newAssignee) {
			changes = append(changes, history.NewOptionalChange(FieldAssignee, currentAssignee, newAssignee))
		}
	}
	if update.DueDate != nil {
		currentDueDate := current.DueDate
		var newDueDate *string
		if *update.DueDate != "" {
			newDueDate = update.DueDate
		}
		if !optionalStringsEqual(currentDueDate, newDueDate) {
			changes = append(changes, history.NewOptionalChange(FieldDueDate, currentDueDate, newDueDate))
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

// auditCreated is the repository audit callback recording a task creation
// inside the mutation transaction.
func (s *Service) auditCreated(ctx context.Context, tx bun.IDB, task *Task) error {
	if recordErr := s.historySvc.Record(ctx, tx, history.EntryBase{
		SubjectType: SubjectTypeTask,
		SubjectID:   task.ID,
		ActorID:     task.AuthorID,
		Action:      history.ActionCreated,
		Changes:     nil,
	}); recordErr != nil {
		return fmt.Errorf("failed to record history: %w", recordErr)
	}

	return nil
}

// auditUpdated returns the repository audit callback recording the applied
// field changes inside the mutation transaction.
func (s *Service) auditUpdated(actorID int64) func(context.Context, bun.IDB, *Task, []history.FieldChange) error {
	return func(ctx context.Context, tx bun.IDB, task *Task, changes []history.FieldChange) error {
		if recordErr := s.historySvc.Record(ctx, tx, history.EntryBase{
			SubjectType: SubjectTypeTask,
			SubjectID:   task.ID,
			ActorID:     actorID,
			Action:      history.ActionUpdated,
			Changes:     changes,
		}); recordErr != nil {
			return fmt.Errorf("failed to record history: %w", recordErr)
		}

		return nil
	}
}
