package tasks_test

import (
	"reflect"
	"testing"

	"github.com/bit-issues/backend/internal/tasks"
)

func TestDeriveChanges(t *testing.T) {
	t.Parallel()

	str := func(v string) *string { return &v }

	older := map[string]*string{
		tasks.FieldTitle:    str("Old title"),
		tasks.FieldStatus:   str("Open"),
		tasks.FieldAssignee: str("3"),
		tasks.FieldDueDate:  str("2026-11-01"),
	}
	newer := map[string]*string{
		tasks.FieldTitle:       str("Old title"), // unchanged - no pair
		tasks.FieldStatus:      str("In Progress"),
		tasks.FieldAssignee:    nil, // cleared
		tasks.FieldDueDate:     str("2026-12-31"),
		tasks.FieldDescription: str("fresh"), // missing in older
	}

	got := tasks.DeriveChanges(newer, older)
	want := []tasks.FieldChange{
		{Field: tasks.FieldDescription, OldValue: nil, NewValue: str("fresh")},
		{Field: tasks.FieldStatus, OldValue: str("Open"), NewValue: str("In Progress")},
		{Field: tasks.FieldAssignee, OldValue: str("3"), NewValue: nil},
		{Field: tasks.FieldDueDate, OldValue: str("2026-11-01"), NewValue: str("2026-12-31")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DeriveChanges mismatch\\n got: %#v\\nwant: %#v", got, want)
	}
}

func TestDeriveChangesNoDifferences(t *testing.T) {
	t.Parallel()

	payload := map[string]*string{tasks.FieldStatus: str("Open")}
	if got := tasks.DeriveChanges(payload, payload); len(got) != 0 {
		t.Errorf("expected no changes, got %#v", got)
	}
}

func str(v string) *string {
	return &v
}
