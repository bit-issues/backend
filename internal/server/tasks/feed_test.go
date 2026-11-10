package tasks_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/comments"
	servertasks "github.com/bit-issues/backend/internal/server/tasks"
	"github.com/bit-issues/backend/internal/tasks"
)

func TestFeedMergeOrder(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.November, 2, 10, 0, 0, 0, time.UTC)
	commentList := []comments.Comment{
		{ID: 10, CreatedAt: base},
		{ID: 11, CreatedAt: base.Add(2 * time.Hour)},
		{ID: 12, CreatedAt: base},
	}
	historyList := []tasks.TaskHistoryEntry{
		{ID: 20, CreatedAt: base.Add(time.Hour)},
		{ID: 21, CreatedAt: base.Add(2 * time.Hour)},
		{ID: 22, CreatedAt: base},
	}

	got := servertasks.FeedPageForTest(commentList, historyList, 0, 20)
	want := []servertasks.FeedEntryView{
		{Kind: "comment", ID: 11, CreatedAt: base.Add(2 * time.Hour)},
		{Kind: "history", ID: 21, CreatedAt: base.Add(2 * time.Hour)},
		{Kind: "history", ID: 20, CreatedAt: base.Add(time.Hour)},
		{Kind: "comment", ID: 12, CreatedAt: base},
		{Kind: "comment", ID: 10, CreatedAt: base},
		{Kind: "history", ID: 22, CreatedAt: base},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("feed mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestFeedPagination(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.November, 2, 10, 0, 0, 0, time.UTC)
	commentList := []comments.Comment{
		{ID: 10, CreatedAt: base},
		{ID: 11, CreatedAt: base.Add(2 * time.Hour)},
	}
	historyList := []tasks.TaskHistoryEntry{
		{ID: 20, CreatedAt: base.Add(time.Hour)},
	}

	got := servertasks.FeedPageForTest(commentList, historyList, 1, 1)
	want := []servertasks.FeedEntryView{
		{Kind: "history", ID: 20, CreatedAt: base.Add(time.Hour)},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("feed page mismatch\n got: %#v\nwant: %#v", got, want)
	}

	empty := servertasks.FeedPageForTest(commentList, historyList, 10, 20)
	if empty == nil {
		t.Fatal("expected non-nil empty page")
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty page, got %#v", empty)
	}
}

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

			got, ok := servertasks.FeedWindowForTest(tt.offset, tt.limit)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("feedWindow(%d, %d) = (%d, %v), want (%d, %v)",
					tt.offset, tt.limit, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
