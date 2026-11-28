package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/feed"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

func toHistoryEntryResponse(
	entry *history.Entry,
	derived []feed.FieldChange,
	usersMap map[int64]users.User,
) clientgo.HistoryEntryResponse {
	changes := make([]clientgo.HistoryChangeResponse, 0, len(derived))
	for _, change := range derived {
		changes = append(changes, clientgo.HistoryChangeResponse{
			Field:    change.Field,
			OldValue: change.OldValue,
			NewValue: change.NewValue,
		})
	}

	return clientgo.HistoryEntryResponse{
		ID:        entry.ID,
		Action:    string(entry.Action),
		Actor:     *dto.ResolveUserBrief(&entry.ActorID, usersMap),
		Changes:   changes,
		CreatedAt: entry.CreatedAt.UTC().Format(time.RFC3339),
	}
}
