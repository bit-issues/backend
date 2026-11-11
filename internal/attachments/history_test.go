package attachments_test

import (
	"testing"

	"github.com/bit-issues/backend/internal/attachments"
)

func TestAttachmentChange(t *testing.T) {
	t.Parallel()

	added := attachments.AttachmentChangeForTest("log.txt", false)
	if added.Field != "attachment" {
		t.Errorf("added.Field = %q, want %q", added.Field, "attachment")
	}
	if added.OldValue != nil {
		t.Errorf("added.OldValue = %v, want nil", *added.OldValue)
	}
	if added.NewValue == nil || *added.NewValue != "log.txt" {
		t.Errorf("added.NewValue = %v, want log.txt", added.NewValue)
	}

	removed := attachments.AttachmentChangeForTest("log.txt", true)
	if removed.Field != "attachment" {
		t.Errorf("removed.Field = %q, want %q", removed.Field, "attachment")
	}
	if removed.OldValue == nil || *removed.OldValue != "log.txt" {
		t.Errorf("removed.OldValue = %v, want log.txt", removed.OldValue)
	}
	if removed.NewValue != nil {
		t.Errorf("removed.NewValue = %v, want nil", *removed.NewValue)
	}
}
