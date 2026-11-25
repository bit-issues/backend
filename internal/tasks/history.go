package tasks

import (
	"strconv"

	"github.com/bit-issues/backend/internal/events"
)

// SubjectTypeTask is the activity-history subject type written for task
// events; history stores subject types as opaque strings and this module
// decides the value.
const SubjectTypeTask = "task"

// Field names recorded in task history payloads. These are the
// task-activity vocabulary: this module writes them (see TaskPayload), the
// feed derives changes from them, and the feed DTO and frontend render them
// - history only stores them as opaque strings.
const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldPriority    = "priority"
	FieldStatus      = "status"
	FieldKind        = "kind"
	FieldAssignee    = "assignee"
	FieldDueDate     = "due_date"
)

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

func formatAssignee(id *int64) *string {
	if id == nil || *id == 0 {
		return nil
	}

	value := strconv.FormatInt(*id, 10)
	return &value
}

// publishCreated broadcasts the task creation after the write has committed.
func (s *Service) publishCreated(task *Task) {
	s.hub.Publish(events.Event{
		Type:      events.TypeTaskCreated,
		TaskID:    task.ID,
		CommentID: 0,
		ActorID:   task.AuthorID,
		Payload:   TaskPayload(task),
	})
}

// publishUpdated broadcasts the task update after the write has committed,
// carrying the actual written values.
func (s *Service) publishUpdated(task *Task, actorID int64) {
	s.hub.Publish(events.Event{
		Type:      events.TypeTaskUpdated,
		TaskID:    task.ID,
		CommentID: 0,
		ActorID:   actorID,
		Payload:   TaskPayload(task),
	})
}
