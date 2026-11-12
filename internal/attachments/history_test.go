package attachments_test

import (
	"testing"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/tasks"
)

func TestAttachmentPayloadCarriesStateAndFile(t *testing.T) {
	t.Parallel()

	task := &tasks.Task{
		ID:          7,
		Title:       "Fix login",
		Status:      tasks.StatusOpen,
		AssigneeID:  ptr[int64](3),
		Description: "steps",
		Priority:    tasks.PriorityMinor,
		Kind:        tasks.KindBug,
	}
	payload := attachments.AttachmentPayloadForTest(task, &attachments.Attachment{
		FileName:  "log.txt",
		SizeBytes: 4096,
	})

	if got := payload[attachments.FieldAttachment]; got == nil || *got != "log.txt" {
		t.Errorf("payload[%q] = %v, want log.txt", attachments.FieldAttachment, got)
	}
	if got := payload[attachments.FieldAttachmentSize]; got == nil || *got != "4096" {
		t.Errorf("payload[%q] = %v, want 4096", attachments.FieldAttachmentSize, got)
	}
	if got := payload[tasks.FieldStatus]; got == nil || *got != string(tasks.StatusOpen) {
		t.Errorf("payload[%q] = %v, want Open", tasks.FieldStatus, got)
	}
	if got := payload[tasks.FieldAssignee]; got == nil || *got != "3" {
		t.Errorf("payload[%q] = %v, want 3", tasks.FieldAssignee, got)
	}
	if len(payload) != 9 {
		t.Errorf("len(payload) = %d, want 9 (7 tracked fields + file + size)", len(payload))
	}
}

func ptr[T any](value T) *T {
	return &value
}
