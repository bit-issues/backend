package feed_test

import (
	"reflect"
	"testing"

	"github.com/bit-issues/backend/internal/feed"
)

func TestDeriveChanges(t *testing.T) {
	t.Parallel()

	older := map[string]*string{
		"title":    str("Old title"),
		"status":   str("Open"),
		"assignee": str("3"),
		"due_date": str("2026-11-01"),
	}
	newer := map[string]*string{
		"title":       str("Old title"), // unchanged - no pair
		"status":      str("In Progress"),
		"assignee":    nil, // cleared
		"due_date":    str("2026-12-31"),
		"description": str("fresh"), // missing in older
	}

	got := feed.DeriveChangesForTest(newer, older)
	// Sorted-key order: assignee, description, due_date, status.
	want := []feed.FieldChange{
		{Field: "assignee", OldValue: str("3"), NewValue: nil},
		{Field: "description", OldValue: nil, NewValue: str("fresh")},
		{Field: "due_date", OldValue: str("2026-11-01"), NewValue: str("2026-12-31")},
		{Field: "status", OldValue: str("Open"), NewValue: str("In Progress")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DeriveChanges mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestDeriveChangesNoDifferences(t *testing.T) {
	t.Parallel()

	payload := map[string]*string{"status": str("Open")}
	if got := feed.DeriveChangesForTest(payload, payload); len(got) != 0 {
		t.Errorf("expected no changes, got %#v", got)
	}
}

func str(v string) *string {
	return &v
}
