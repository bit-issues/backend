package history_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/events"
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

func TestEntryFromEventMapsHistoryTypes(t *testing.T) {
	t.Parallel()

	payload := map[string]*string{"status": ptr("Open")}

	tests := []struct {
		name   string
		event  events.Event
		want   history.EntryBase
		wantOK bool
	}{
		{
			name:  "task created",
			event: events.Event{Type: events.TypeTaskCreated, TaskID: 7, ActorID: 3, Payload: payload},
			want: history.EntryBase{
				SubjectType: "task",
				SubjectID:   7,
				ActorID:     3,
				Action:      history.ActionCreated,
				Payload:     payload,
			},
			wantOK: true,
		},
		{
			name:  "task updated",
			event: events.Event{Type: events.TypeTaskUpdated, TaskID: 7, ActorID: 3, Payload: payload},
			want: history.EntryBase{
				SubjectType: "task",
				SubjectID:   7,
				ActorID:     3,
				Action:      history.ActionUpdated,
				Payload:     payload,
			},
			wantOK: true,
		},
		{
			name:  "attachment added",
			event: events.Event{Type: events.TypeAttachmentAdded, TaskID: 7, ActorID: 3, Payload: payload},
			want: history.EntryBase{
				SubjectType: "task",
				SubjectID:   7,
				ActorID:     3,
				Action:      history.ActionAttachmentAdded,
				Payload:     payload,
			},
			wantOK: true,
		},
		{
			name:  "attachment removed",
			event: events.Event{Type: events.TypeAttachmentRemoved, TaskID: 7, ActorID: 3, Payload: payload},
			want: history.EntryBase{
				SubjectType: "task",
				SubjectID:   7,
				ActorID:     3,
				Action:      history.ActionAttachmentRemoved,
				Payload:     payload,
			},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := history.EntryFromEventForTest(tt.event)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("entry mismatch\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestEntryFromEventSkipsNonHistoryTypes(t *testing.T) {
	t.Parallel()

	for _, eventType := range []string{
		events.TypeStreamConnected,
		events.TypeHistoryRecorded,
		events.TypeCommentCreated,
		events.TypeCommentUpdated,
		events.TypeCommentDeleted,
	} {
		t.Run(eventType, func(t *testing.T) {
			t.Parallel()

			_, ok := history.EntryFromEventForTest(events.Event{Type: eventType, TaskID: 7})
			if ok {
				t.Errorf("type %q unexpectedly mapped to a history entry", eventType)
			}
		})
	}
}

func ptr[T any](value T) *T {
	return &value
}
