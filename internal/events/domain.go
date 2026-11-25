package events

// Event types broadcast on the events stream.
const (
	TypeStreamConnected   = "stream.connected"
	TypeCommentCreated    = "comment.created"
	TypeCommentUpdated    = "comment.updated"
	TypeCommentDeleted    = "comment.deleted"
	TypeTaskCreated       = "task.created"
	TypeTaskUpdated       = "task.updated"
	TypeAttachmentAdded   = "attachment.added"
	TypeAttachmentRemoved = "attachment.removed"

	// TypeHistoryRecorded signals that an activity row was persisted. Clients
	// refresh history-backed feeds on it instead of racing the async consumer.
	TypeHistoryRecorded = "history.recorded"
)

// Event describes an operation over an entity. Consumers use Type and the
// identifiers to decide what to do; ActorID and Payload carry the same written
// values the activity history needs. The SSE wire format projects this down to
// identifiers only, so clients still refetch the affected resource.
type Event struct {
	Type      string
	TaskID    int64
	CommentID int64
	ActorID   int64
	Payload   map[string]*string
}
