package events

// Event types broadcast on the events stream.
const (
	TypeCommentCreated = "comment.created"
	TypeCommentUpdated = "comment.updated"
	TypeCommentDeleted = "comment.deleted"
)

// Event is a minimal notification about a data change. It carries only
// identifiers; clients refetch the affected resource instead of applying a
// partial payload.
type Event struct {
	Type      string `json:"type"`
	TaskID    int64  `json:"task_id"`
	CommentID int64  `json:"comment_id"`
}
