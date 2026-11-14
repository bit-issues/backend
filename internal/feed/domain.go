package feed

import (
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/users"
)

// FieldChange is one field-level change derived at read time from recorded
// value payloads. Field names the payload key that changed; values are
// rendered for display (assignee ids resolved to names). History never sees
// this type, and it carries no wire format - the feed DTO maps it at the
// transport boundary.
type FieldChange struct {
	Field    string
	OldValue *string
	NewValue *string
}

// Page is one page of the task activity feed: page-sized entries with their
// derived change pairs (display-ready values), the assignee IDs referenced
// anywhere in the fetched window, the users needed to render actors and
// assignees, and the subject's total entry count.
type Page struct {
	Entries     []history.Entry
	Changes     [][]FieldChange
	AssigneeIDs []int64
	Users       map[int64]users.User
	Total       int
}
