package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/tasks"
)

// FeedEntryView is a test-only projection of a feed entry.
type FeedEntryView struct {
	Kind      string
	ID        int64
	CreatedAt time.Time
}

// FeedPageForTest exposes mergeFeedEntries and pageFeedEntries to the external test package.
func FeedPageForTest(
	commentList []comments.Comment,
	historyList []tasks.TaskHistoryEntry,
	offset, limit int,
) []FeedEntryView {
	entries := pageFeedEntries(mergeFeedEntries(commentList, historyList), offset, limit)
	views := make([]FeedEntryView, 0, len(entries))
	for i := range entries {
		views = append(views, FeedEntryView{
			Kind:      string(entries[i].Kind),
			ID:        entries[i].ID,
			CreatedAt: entries[i].CreatedAt,
		})
	}

	return views
}
