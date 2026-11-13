package tasks

import (
	"github.com/bit-issues/backend/internal/feed"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
)

// FeedQuery represents pagination query parameters for the task feed.
type FeedQuery struct {
	dto.PaginationQuery
}

// FeedResponse represents a paginated task activity feed.
//
//	@Description	Paginated activity feed with total count.
type FeedResponse struct {
	Items []HistoryEntryResponse `json:"items"`
	Total int                    `json:"total"`
}

func toFeedResponse(
	entries []history.Entry,
	changes [][]feed.FieldChange,
	usersMap map[int64]users.User,
	total int,
) *FeedResponse {
	items := make([]HistoryEntryResponse, 0, len(entries))
	for i := range entries {
		items = append(items, toHistoryEntryResponse(&entries[i], changes[i], usersMap))
	}

	return &FeedResponse{
		Items: items,
		Total: total,
	}
}
