package attachments

import (
	"context"
	"strconv"

	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"go.uber.org/zap"
)

// FieldAttachment is the activity-feed payload key under which attachment
// events carry the file name; this module is its only producer and the feed
// reader is its only consumer.
const FieldAttachment = "attachment"

// FieldAttachmentSize is the activity-feed payload key under which attachment
// events carry the uploaded size in bytes (decimal string).
const FieldAttachmentSize = "size_bytes"

// attachmentPayload carries the task's actual written values (the snapshot
// chain must stay unbroken for read-side derivation) plus the file name and
// size.
func attachmentPayload(task *tasks.Task, attachment *Attachment) map[string]*string {
	payload := tasks.TaskPayload(task)
	name := attachment.FileName
	payload[FieldAttachment] = &name
	size := strconv.FormatUint(attachment.SizeBytes, 10)
	payload[FieldAttachmentSize] = &size

	return payload
}

// recordAdded best-effort appends the upload event after the attachment write
// committed; failures are logged, never surfaced.
func (s *Service) recordAdded(
	ctx context.Context,
	task *tasks.Task,
	uploaderID int64,
	attachment *Attachment,
) {
	if err := s.historySvc.Record(ctx, history.EntryBase{
		SubjectType: tasks.SubjectTypeTask,
		SubjectID:   task.ID,
		ActorID:     uploaderID,
		Action:      history.ActionAttachmentAdded,
		Payload:     attachmentPayload(task, attachment),
	}); err != nil {
		s.logger.Warn("failed to record history entry",
			zap.String("action", string(history.ActionAttachmentAdded)),
			zap.Int64("subject_id", task.ID),
			zap.String("file_name", attachment.FileName),
			zap.Error(err),
		)
	}
}

// recordRemoved best-effort appends the deletion event after the attachment
// write committed; failures are logged, never surfaced.
func (s *Service) recordRemoved(
	ctx context.Context,
	task *tasks.Task,
	actorID int64,
	attachment *Attachment,
) {
	if err := s.historySvc.Record(ctx, history.EntryBase{
		SubjectType: tasks.SubjectTypeTask,
		SubjectID:   task.ID,
		ActorID:     actorID,
		Action:      history.ActionAttachmentRemoved,
		Payload:     attachmentPayload(task, attachment),
	}); err != nil {
		s.logger.Warn("failed to record history entry",
			zap.String("action", string(history.ActionAttachmentRemoved)),
			zap.Int64("subject_id", task.ID),
			zap.String("file_name", attachment.FileName),
			zap.Error(err),
		)
	}
}
