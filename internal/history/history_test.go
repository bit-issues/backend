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
		Payload:     map[string]*string{},
	}, now)

	want := history.Entry{
		EntryBase: history.EntryBase{
			SubjectType: "task",
			SubjectID:   7,
			ActorID:     3,
			Action:      history.ActionCreated,
			Payload:     map[string]*string{},
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
	payload := map[string]*string{"title": ptr("b")}
	got := history.EntryFromBaseForTest(history.EntryBase{
		SubjectType: "task",
		SubjectID:   7,
		ActorID:     3,
		Action:      history.ActionUpdated,
		Payload:     payload,
	}, now)

	if !reflect.DeepEqual(got.Payload, payload) {
		t.Errorf("Payload = %#v, want %#v", got.Payload, payload)
	}
	if got.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func ptr[T any](value T) *T {
	return &value
}
