package feed_test

import (
	"reflect"
	"testing"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/feed"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
)

func TestFeedWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		offset int
		limit  int
		want   int
		wantOK bool
	}{
		{"first page", 0, 20, 20, true},
		{"within bound", 4900, 100, 5000, true},
		{"exactly at bound", 0, 5000, 5000, true},
		{"one past bound", 1, 5000, 0, false},
		{"huge offset", 1 << 40, 20, 0, false},
		{"overflow offset", 9223372036854775807, 20, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := feed.FeedWindowForTest(tt.offset, tt.limit)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("feedWindow(%d, %d) = (%d, %v), want (%d, %v)",
					tt.offset, tt.limit, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDeriveHistoryChanges(t *testing.T) {
	t.Parallel()

	state := func(status string) map[string]*string {
		return map[string]*string{tasks.FieldStatus: str(status)}
	}
	withFile := func(payload map[string]*string, value string) map[string]*string {
		payload[attachments.FieldAttachment] = &value
		return payload
	}
	withSize := func(payload map[string]*string, value string) map[string]*string {
		payload[attachments.FieldAttachmentSize] = &value
		return payload
	}

	entries := []history.Entry{
		{EntryBase: history.EntryBase{Action: history.ActionUpdated, Payload: state("In Progress")}},
		{EntryBase: history.EntryBase{
			Action:  history.ActionAttachmentAdded,
			Payload: withSize(withFile(state("Open"), "a.txt"), "10"),
		}},
		{EntryBase: history.EntryBase{
			Action:  history.ActionAttachmentRemoved,
			Payload: withSize(withFile(state("Open"), "b.txt"), "20"),
		}},
		{EntryBase: history.EntryBase{
			Action:  history.ActionAttachmentRemoved,
			Payload: withFile(state("Open"), "c.txt"),
		}},
		{EntryBase: history.EntryBase{Action: history.ActionCreated, Payload: state("Open")}},
	}

	got := feed.DeriveHistoryChangesForTest(entries)
	want := [][]feed.FieldChange{
		{{Field: tasks.FieldStatus, OldValue: str("Open"), NewValue: str("In Progress")}},
		{
			{Field: attachments.FieldAttachment, OldValue: nil, NewValue: str("a.txt")},
			{Field: attachments.FieldAttachmentSize, OldValue: nil, NewValue: str("10")},
		},
		{
			{Field: attachments.FieldAttachment, OldValue: str("b.txt"), NewValue: nil},
			{Field: attachments.FieldAttachmentSize, OldValue: str("20"), NewValue: nil},
		},
		{{Field: attachments.FieldAttachment, OldValue: str("c.txt"), NewValue: nil}},
		nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("derive mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestResolveChangeValues(t *testing.T) {
	t.Parallel()

	usersMap := map[int64]users.User{7: {ID: 7, Name: "Alice"}}

	changes := [][]feed.FieldChange{
		{
			{Field: tasks.FieldAssignee, OldValue: str("7"), NewValue: str("42")},
			{Field: tasks.FieldAssignee, OldValue: str("not-a-number"), NewValue: nil},
			{Field: tasks.FieldStatus, OldValue: str("Open"), NewValue: str("In Progress")},
		},
		nil,
	}

	got := feed.ResolveChangeValuesForTest(changes, usersMap)
	want := [][]feed.FieldChange{
		{
			// Known user renders by name; unknown falls back to a placeholder;
			// unparseable and non-assignee values pass through; nil stays nil.
			{Field: tasks.FieldAssignee, OldValue: str("Alice"), NewValue: str("#42")},
			{Field: tasks.FieldAssignee, OldValue: str("not-a-number"), NewValue: nil},
			{Field: tasks.FieldStatus, OldValue: str("Open"), NewValue: str("In Progress")},
		},
		nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("resolve mismatch\n got: %#v\nwant: %#v", got, want)
	}

	if gotNil := feed.ResolveChangeValuesForTest(nil, usersMap); gotNil != nil {
		t.Errorf("expected nil for nil changes, got %#v", gotNil)
	}
}
