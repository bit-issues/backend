package attachments

import (
	"strconv"

	"github.com/bit-issues/backend/internal/events"
	"github.com/bit-issues/backend/internal/tasks"
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

// publishAdded broadcasts the confirmed upload after the write has committed.
func (s *Service) publishAdded(task *tasks.Task, uploaderID int64, attachment *Attachment) {
	s.hub.Publish(events.Event{
		Type:      events.TypeAttachmentAdded,
		TaskID:    task.ID,
		CommentID: 0,
		ActorID:   uploaderID,
		Payload:   attachmentPayload(task, attachment),
	})
}

// publishRemoved broadcasts the attachment deletion after the write has
// committed.
func (s *Service) publishRemoved(task *tasks.Task, actorID int64, attachment *Attachment) {
	s.hub.Publish(events.Event{
		Type:      events.TypeAttachmentRemoved,
		TaskID:    task.ID,
		CommentID: 0,
		ActorID:   actorID,
		Payload:   attachmentPayload(task, attachment),
	})
}
