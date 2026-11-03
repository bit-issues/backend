package tasks_test

import (
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/tasks"
)

func TestNewTaskImportZeroCreatedAtFallsBackToNow(t *testing.T) {
	t.Parallel()

	before := time.Now()
	got := tasks.NewTaskImportForTest(tasks.Task{})
	after := time.Now()

	if got.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt fallback to now, got zero value")
	}
	if got.CreatedAt.Before(before) || got.CreatedAt.After(after) {
		t.Fatalf("expected CreatedAt within [%v, %v], got %v", before, after, got.CreatedAt)
	}
}

func TestNewTaskImportPreservesCreatedAt(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, time.May, 4, 3, 2, 1, 0, time.UTC)
	got := tasks.NewTaskImportForTest(tasks.Task{CreatedAt: createdAt})

	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected imported CreatedAt %v, got %v", createdAt, got.CreatedAt)
	}
}
