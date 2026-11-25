package history

import (
	"context"

	"github.com/bit-issues/backend/internal/events"
	"go.uber.org/zap"
)

// subjectTypeTask is the subject type task and attachment events are recorded
// under; it matches the value task-facing reads query by.
const subjectTypeTask = "task"

// Consumer records activity history for operations published on the event
// bus. It is a lossy subscriber: events overflowing its buffer are dropped by
// the hub, matching the activity feed's best-effort semantics.
type Consumer struct {
	hub        *events.Hub
	historySvc *Service
	logger     *zap.Logger
}

// NewConsumer creates a history consumer for the given hub and service.
func NewConsumer(hub *events.Hub, historySvc *Service, logger *zap.Logger) *Consumer {
	return &Consumer{
		hub:        hub,
		historySvc: historySvc,
		logger:     logger,
	}
}

// Run consumes events until the context is cancelled or the hub closes.
func (c *Consumer) Run(ctx context.Context) error {
	sub, cancel := c.hub.Subscribe()
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-sub:
			if !ok {
				return nil
			}
			c.record(ctx, ev)
		}
	}
}

func (c *Consumer) record(ctx context.Context, ev events.Event) {
	entry, ok := entryFromEvent(ev)
	if !ok {
		return
	}

	if err := c.historySvc.Record(ctx, entry); err != nil {
		c.logger.Warn("failed to record history entry from event",
			zap.String("type", ev.Type),
			zap.Int64("subject_id", entry.SubjectID),
			zap.Error(err),
		)
		return
	}

	c.hub.Publish(events.Event{
		Type:      events.TypeHistoryRecorded,
		TaskID:    ev.TaskID,
		CommentID: 0,
		ActorID:   0,
		Payload:   nil,
	})
}

// entryFromEvent maps an operation to the history row it produces. Types with
// no history representation (stream bookkeeping and comments, which are added
// in a later step) are skipped.
func entryFromEvent(ev events.Event) (EntryBase, bool) {
	var action Action

	switch ev.Type {
	case events.TypeTaskCreated:
		action = ActionCreated
	case events.TypeTaskUpdated:
		action = ActionUpdated
	case events.TypeAttachmentAdded:
		action = ActionAttachmentAdded
	case events.TypeAttachmentRemoved:
		action = ActionAttachmentRemoved
	default:
		return EntryBase{}, false
	}

	return EntryBase{
		SubjectType: subjectTypeTask,
		SubjectID:   ev.TaskID,
		ActorID:     ev.ActorID,
		Action:      action,
		Payload:     ev.Payload,
	}, true
}
