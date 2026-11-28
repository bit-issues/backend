package tasks

import (
	"github.com/bit-issues/backend/internal/feed"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

func toFeedResponse(
	entries []history.Entry,
	changes [][]feed.FieldChange,
	usersMap map[int64]users.User,
	total int,
) *clientgo.FeedResponse {
	items := make([]clientgo.HistoryEntryResponse, 0, len(entries))
	for i := range entries {
		items = append(items, toHistoryEntryResponse(&entries[i], changes[i], usersMap))
	}

	return &clientgo.FeedResponse{
		Items: items,
		Total: total,
	}
}
