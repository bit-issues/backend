package history_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/history"
)

func TestEntryFromBaseStampsCreatedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	got := history.EntryFromBaseForTest(history.EntryBase{
		SubjectType: "task",
		SubjectID:   7,
		ActorID:     3,
		Action:      history.ActionCreated,
	}, now)

	want := history.Entry{
		EntryBase: history.EntryBase{
			SubjectType: "task",
			SubjectID:   7,
			ActorID:     3,
			Action:      history.ActionCreated,
		},
		CreatedAt: now,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("entry mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestEntryFromBasePassesChangesThrough(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	changes := []history.FieldChange{{Field: "title", OldValue: ptr("a"), NewValue: ptr("b")}}
	got := history.EntryFromBaseForTest(history.EntryBase{
		SubjectType: "task",
		SubjectID:   7,
		ActorID:     3,
		Action:      history.ActionUpdated,
		Changes:     changes,
	}, now)

	if !reflect.DeepEqual(got.Changes, changes) {
		t.Errorf("Changes = %#v, want %#v", got.Changes, changes)
	}
	if got.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func ptr[T any](value T) *T {
	return &value
}
