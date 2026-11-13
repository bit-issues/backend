package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/feed"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
)

// HistoryChangeResponse represents a single field change in task history.
//
//	@Description	Field-level task change with old and new values.
type HistoryChangeResponse struct {
	Field    string  `json:"field"`
	OldValue *string `json:"old_value"`
	NewValue *string `json:"new_value"`
}

// HistoryEntryResponse represents a single task history entry.
//
//	@Description	Task history entry with actor and field changes.
type HistoryEntryResponse struct {
	ID        int64                   `json:"id"`
	Action    string                  `json:"action"`
	Actor     dto.UserBrief           `json:"actor"`
	Changes   []HistoryChangeResponse `json:"changes"`
	CreatedAt string                  `json:"created_at"`
}

func toHistoryEntryResponse(
	entry *history.Entry,
	derived []feed.FieldChange,
	usersMap map[int64]users.User,
) HistoryEntryResponse {
	changes := make([]HistoryChangeResponse, 0, len(derived))
	for _, change := range derived {
		changes = append(changes, HistoryChangeResponse{
			Field:    change.Field,
			OldValue: change.OldValue,
			NewValue: change.NewValue,
		})
	}

	return HistoryEntryResponse{
		ID:        entry.ID,
		Action:    string(entry.Action),
		Actor:     *dto.ResolveUserBrief(&entry.ActorID, usersMap),
		Changes:   changes,
		CreatedAt: entry.CreatedAt.UTC().Format(time.RFC3339),
	}
}
