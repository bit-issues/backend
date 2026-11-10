package tasks_test

import (
	"reflect"
	"testing"

	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
)

func TestDiffTaskAllFields(t *testing.T) {
	t.Parallel()

	current := &tasks.Task{
		Title:       "Old title",
		Description: "Old description",
		Priority:    tasks.PriorityMajor,
		Status:      tasks.StatusNew,
		Kind:        tasks.KindBug,
		AssigneeID:  ptr[int64](7),
		DueDate:     ptr("2026-11-01"),
	}
	update := tasks.TaskUpdate{
		Title:       ptr("New title"),
		Description: ptr("New description"),
		Priority:    ptr(tasks.PriorityCritical),
		Status:      ptr(tasks.StatusInProgress),
		Kind:        ptr(tasks.KindTask),
		AssigneeID:  ptr[int64](8),
		DueDate:     ptr("2026-12-31"),
	}

	got := tasks.DiffTaskForTest(current, update)
	want := []history.FieldChange{
		{Field: history.FieldTitle, OldValue: ptr("Old title"), NewValue: ptr("New title")},
		{
			Field:    history.FieldDescription,
			OldValue: ptr("Old description"),
			NewValue: ptr("New description"),
		},
		{Field: history.FieldPriority, OldValue: ptr("Major"), NewValue: ptr("Critical")},
		{Field: history.FieldStatus, OldValue: ptr("New"), NewValue: ptr("In Progress")},
		{Field: history.FieldKind, OldValue: ptr("Bug"), NewValue: ptr("Task")},
		{Field: history.FieldAssignee, OldValue: ptr("7"), NewValue: ptr("8")},
		{Field: history.FieldDueDate, OldValue: ptr("2026-11-01"), NewValue: ptr("2026-12-31")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diff mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestDiffTaskNoChanges(t *testing.T) {
	t.Parallel()

	current := &tasks.Task{
		Title:       "Same title",
		Description: "Same description",
		Priority:    tasks.PriorityMajor,
		Status:      tasks.StatusOpen,
		Kind:        tasks.KindBug,
		AssigneeID:  ptr[int64](7),
		DueDate:     ptr("2026-11-01"),
	}
	update := tasks.TaskUpdate{
		Title:       ptr("Same title"),
		Description: ptr("Same description"),
		Priority:    ptr(tasks.PriorityMajor),
		Status:      ptr(tasks.StatusOpen),
		Kind:        ptr(tasks.KindBug),
		AssigneeID:  ptr[int64](7),
		DueDate:     ptr("2026-11-01"),
	}

	got := tasks.DiffTaskForTest(current, update)
	if len(got) != 0 {
		t.Fatalf("expected no changes, got %#v", got)
	}
}

func TestDiffTaskClearsOptionalFields(t *testing.T) {
	t.Parallel()

	current := &tasks.Task{
		AssigneeID: ptr[int64](7),
		DueDate:    ptr("2026-11-01"),
	}
	update := tasks.TaskUpdate{
		AssigneeID: ptr[int64](0),
		DueDate:    ptr(""),
	}

	got := tasks.DiffTaskForTest(current, update)
	want := []history.FieldChange{
		{Field: history.FieldAssignee, OldValue: ptr("7"), NewValue: nil},
		{Field: history.FieldDueDate, OldValue: ptr("2026-11-01"), NewValue: nil},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diff mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestDiffTaskSetsOptionalFields(t *testing.T) {
	t.Parallel()

	current := &tasks.Task{}
	update := tasks.TaskUpdate{
		AssigneeID: ptr[int64](9),
		DueDate:    ptr("2026-12-31"),
	}

	got := tasks.DiffTaskForTest(current, update)
	want := []history.FieldChange{
		{Field: history.FieldAssignee, OldValue: nil, NewValue: ptr("9")},
		{Field: history.FieldDueDate, OldValue: nil, NewValue: ptr("2026-12-31")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diff mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func ptr[T any](value T) *T {
	return &value
}
