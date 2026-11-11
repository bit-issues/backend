package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/history"
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
	historyList []history.Entry,
	offset, limit int,
) []FeedEntryView {
	historyChanges := make([][]tasks.FieldChange, len(historyList))
	entries := pageFeedEntries(mergeFeedEntries(commentList, historyList, historyChanges), offset, limit)
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

// DeriveHistoryChangesForTest exposes deriveHistoryChanges to the external test package.
var DeriveHistoryChangesForTest = deriveHistoryChanges

// FeedWindowForTest exposes feedWindow to the external test package.
func FeedWindowForTest(offset, limit int) (int, bool) {
	return feedWindow(offset, limit)
}
