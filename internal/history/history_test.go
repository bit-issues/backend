package history_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/history"
)

func TestNewAttachmentAddedEntry(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	got := history.NewAttachmentAddedEntry(7, 3, "log.txt", at)

	want := history.Entry{
		TaskID:    7,
		ActorID:   3,
		Action:    history.ActionAttachmentAdded,
		Changes:   []history.FieldChange{{Field: history.FieldAttachment, OldValue: nil, NewValue: ptr("log.txt")}},
		CreatedAt: at,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewAttachmentAddedEntry mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestNewAttachmentRemovedEntry(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	got := history.NewAttachmentRemovedEntry(7, 3, "log.txt", at)

	want := history.Entry{
		TaskID:    7,
		ActorID:   3,
		Action:    history.ActionAttachmentRemoved,
		Changes:   []history.FieldChange{{Field: history.FieldAttachment, OldValue: ptr("log.txt"), NewValue: nil}},
		CreatedAt: at,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewAttachmentRemovedEntry mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func ptr[T any](value T) *T {
	return &value
}
