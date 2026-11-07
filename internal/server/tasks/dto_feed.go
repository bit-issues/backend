package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
)

// FeedQuery represents pagination query parameters for the task feed.
type FeedQuery struct {
	dto.PaginationQuery
}

// FeedItemResponse represents a single comment or history entry in the task feed.
//
//	@Description	Activity feed item with exactly one of comment or history populated.
type FeedItemResponse struct {
	Type      string                `json:"type"`
	ID        int64                 `json:"id"`
	CreatedAt string                `json:"created_at"`
	Comment   *CommentResponse      `json:"comment,omitempty"`
	History   *HistoryEntryResponse `json:"history,omitempty"`
}

// FeedResponse represents a paginated task activity feed.
//
//	@Description	Paginated activity feed with total count.
type FeedResponse struct {
	Items []FeedItemResponse `json:"items"`
	Total int                `json:"total"`
}

func toFeedResponse(
	entries []feedEntry,
	usersMap map[int64]users.User,
	total int,
) *FeedResponse {
	items := make([]FeedItemResponse, 0, len(entries))

	for i := range entries {
		entry := &entries[i]
		createdAt := entry.CreatedAt.UTC().Format(time.RFC3339)

		if entry.Kind == feedEntryKindComment && entry.Comment != nil {
			comment := toCommentResponse(entry.Comment, usersMap)
			items = append(items, FeedItemResponse{
				Type:      string(entry.Kind),
				ID:        entry.ID,
				CreatedAt: createdAt,
				Comment:   &comment,
				History:   nil,
			})
			continue
		}

		if entry.Kind == feedEntryKindHistory && entry.History != nil {
			history := toHistoryEntryResponse(entry.History, usersMap)
			items = append(items, FeedItemResponse{
				Type:      string(entry.Kind),
				ID:        entry.ID,
				CreatedAt: createdAt,
				Comment:   nil,
				History:   &history,
			})
		}
	}

	return &FeedResponse{
		Items: items,
		Total: total,
	}
}
