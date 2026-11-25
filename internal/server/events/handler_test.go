package events_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/bit-issues/backend/internal/events"
	serverevents "github.com/bit-issues/backend/internal/server/events"
)

func TestWriteEventProjectsIdentifiersOnly(t *testing.T) {
	t.Parallel()

	status := "In Progress"
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	err := serverevents.WriteEventForTest(writer, events.Event{
		Type:      events.TypeTaskUpdated,
		TaskID:    7,
		CommentID: 3,
		ActorID:   42,
		Payload:   map[string]*string{"status": &status},
	})
	if err != nil {
		t.Fatalf("writeEvent: %v", err)
	}

	want := "event: task.updated\n" +
		"data: {\"type\":\"task.updated\",\"task_id\":7,\"comment_id\":3}\n\n"
	if got := buf.String(); got != want {
		t.Errorf("wire frame mismatch\n got: %q\nwant: %q", got, want)
	}
}
