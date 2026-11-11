package attachments

import (
	"context"
	"fmt"

	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/uptrace/bun"
)

// fieldAttachment is the activity-feed field name under which attachment
// events are recorded; this module is its only producer.
const fieldAttachment = "attachment"

// attachmentChange builds the history change for an attachment event: the
// file name lands in NewValue for additions and in OldValue for removals,
// mirroring before/after state semantics. Field names are opaque to history,
// so this module constructs the payload itself.
func attachmentChange(fileName string, removed bool) history.FieldChange {
	name := fileName
	if removed {
		return history.FieldChange{Field: fieldAttachment, OldValue: &name, NewValue: nil}
	}

	return history.FieldChange{Field: fieldAttachment, OldValue: nil, NewValue: &name}
}

// auditAdded returns the repository audit callback recording a confirmed
// upload inside the mutation transaction.
func (s *Service) auditAdded(uploaderID int64) history.AuditFunc[*Attachment] {
	return func(ctx context.Context, tx bun.IDB, att *Attachment) error {
		if recordErr := s.historySvc.Record(ctx, tx, history.EntryBase{
			SubjectType: tasks.SubjectTypeTask,
			SubjectID:   att.TaskID,
			ActorID:     uploaderID,
			Action:      history.ActionAttachmentAdded,
			Changes:     []history.FieldChange{attachmentChange(att.FileName, false)},
		}); recordErr != nil {
			return fmt.Errorf("failed to record history: %w", recordErr)
		}

		return nil
	}
}

// auditRemoved returns the repository audit callback recording a deletion
// inside the mutation transaction.
func (s *Service) auditRemoved(actorID int64) history.AuditFunc[*Attachment] {
	return func(ctx context.Context, tx bun.IDB, att *Attachment) error {
		if recordErr := s.historySvc.Record(ctx, tx, history.EntryBase{
			SubjectType: tasks.SubjectTypeTask,
			SubjectID:   att.TaskID,
			ActorID:     actorID,
			Action:      history.ActionAttachmentRemoved,
			Changes:     []history.FieldChange{attachmentChange(att.FileName, true)},
		}); recordErr != nil {
			return fmt.Errorf("failed to record history: %w", recordErr)
		}

		return nil
	}
}
