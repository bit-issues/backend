package tasks

import (
	"context"
	"strconv"

	"github.com/bit-issues/backend/internal/history"
	"go.uber.org/zap"
)

// FieldChange is one field-level change derived at read time from recorded
// value payloads. The Field* vocabulary above names the fields; history never
// sees this type.
type FieldChange struct {
	Field    string  `json:"field"`
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

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

// TaskPayload snapshots the tracked fields of a task as written values. The
// keys mirror the Field* vocabulary so readers can derive changes by
// comparing consecutive snapshots.
func TaskPayload(task *Task) map[string]*string {
	title := task.Title
	description := task.Description
	priority := string(task.Priority)
	status := string(task.Status)
	kind := string(task.Kind)

	return map[string]*string{
		FieldTitle:       &title,
		FieldDescription: &description,
		FieldPriority:    &priority,
		FieldStatus:      &status,
		FieldKind:        &kind,
		FieldAssignee:    formatAssignee(task.AssigneeID),
		FieldDueDate:     task.DueDate,
	}
}

// DeriveChanges compares a newer payload against its next-older snapshot and
// returns the fields that differ, in trackedFields order.
func DeriveChanges(newer, older map[string]*string) []FieldChange {
	// Derivation order doubles as wire order so renders stay deterministic.
	trackedFields := []string{
		FieldTitle,
		FieldDescription,
		FieldPriority,
		FieldStatus,
		FieldKind,
		FieldAssignee,
		FieldDueDate,
	}
	changes := make([]FieldChange, 0, len(trackedFields))

	for _, field := range trackedFields {
		newValue, ok := newer[field]
		if !ok {
			continue
		}
		oldValue := older[field]
		if !optionalStringsEqual(oldValue, newValue) {
			changes = append(changes, newOptionalChange(field, oldValue, newValue))
		}
	}

	return changes
}

// diffTask computes the field-level changes between the current task state
// and an update. An empty result means the update is a no-op.
func diffTask(current *Task, update TaskUpdate) []FieldChange {
	changes := make([]FieldChange, 0)

	if update.Title != nil && *update.Title != current.Title {
		changes = append(changes, newStringChange(FieldTitle, current.Title, *update.Title))
	}
	if update.Description != nil && *update.Description != current.Description {
		changes = append(
			changes,
			newStringChange(FieldDescription, current.Description, *update.Description),
		)
	}
	if update.Priority != nil && *update.Priority != current.Priority {
		changes = append(
			changes,
			newStringChange(FieldPriority, string(current.Priority), string(*update.Priority)),
		)
	}
	if update.Status != nil && *update.Status != current.Status {
		changes = append(
			changes,
			newStringChange(FieldStatus, string(current.Status), string(*update.Status)),
		)
	}
	if update.Kind != nil && *update.Kind != current.Kind {
		changes = append(
			changes,
			newStringChange(FieldKind, string(current.Kind), string(*update.Kind)),
		)
	}
	if update.AssigneeID != nil {
		currentAssignee := formatAssignee(current.AssigneeID)
		newAssignee := formatAssignee(update.AssigneeID)
		if !optionalStringsEqual(currentAssignee, newAssignee) {
			changes = append(changes, newOptionalChange(FieldAssignee, currentAssignee, newAssignee))
		}
	}
	if update.DueDate != nil {
		currentDueDate := current.DueDate
		var newDueDate *string
		if *update.DueDate != "" {
			newDueDate = update.DueDate
		}
		if !optionalStringsEqual(currentDueDate, newDueDate) {
			changes = append(changes, newOptionalChange(FieldDueDate, currentDueDate, newDueDate))
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

// recordCreated best-effort appends the creation event after the task write
// has committed; failures are logged, never surfaced - the task matters, the
// feed does not.
func (s *Service) recordCreated(ctx context.Context, task *Task) {
	if err := s.historySvc.Record(ctx, history.EntryBase{
		SubjectType: SubjectTypeTask,
		SubjectID:   task.ID,
		ActorID:     task.AuthorID,
		Action:      history.ActionCreated,
		Payload:     TaskPayload(task),
	}); err != nil {
		s.logger.Warn("failed to record history entry",
			zap.String("action", string(history.ActionCreated)),
			zap.Int64("subject_id", task.ID),
			zap.Error(err),
		)
	}
}

// recordUpdated best-effort appends the update event after the write
// committed, carrying the actual written values.
func (s *Service) recordUpdated(ctx context.Context, task *Task, actorID int64) {
	if err := s.historySvc.Record(ctx, history.EntryBase{
		SubjectType: SubjectTypeTask,
		SubjectID:   task.ID,
		ActorID:     actorID,
		Action:      history.ActionUpdated,
		Payload:     TaskPayload(task),
	}); err != nil {
		s.logger.Warn("failed to record history entry",
			zap.String("action", string(history.ActionUpdated)),
			zap.Int64("subject_id", task.ID),
			zap.Error(err),
		)
	}
}
