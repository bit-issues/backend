package tasks

import (
	"slices"
	"time"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/tasks"
)

type feedEntryKind string

const (
	feedEntryKindComment feedEntryKind = "comment"
	feedEntryKindHistory feedEntryKind = "history"
)

type feedEntry struct {
	Kind      feedEntryKind
	ID        int64
	CreatedAt time.Time
	Comment   *comments.Comment
	History   *tasks.TaskHistoryEntry
}

func newCommentFeedEntry(comment *comments.Comment) feedEntry {
	return feedEntry{
		Kind:      feedEntryKindComment,
		ID:        comment.ID,
		CreatedAt: comment.CreatedAt,
		Comment:   comment,
		History:   nil,
	}
}

func newHistoryFeedEntry(history *tasks.TaskHistoryEntry) feedEntry {
	return feedEntry{
		Kind:      feedEntryKindHistory,
		ID:        history.ID,
		CreatedAt: history.CreatedAt,
		Comment:   nil,
		History:   history,
	}
}

func mergeFeedEntries(
	commentList []comments.Comment,
	historyList []tasks.TaskHistoryEntry,
) []feedEntry {
	entries := make([]feedEntry, 0, len(commentList)+len(historyList))

	for i := range commentList {
		entries = append(entries, newCommentFeedEntry(&commentList[i]))
	}
	for i := range historyList {
		entries = append(entries, newHistoryFeedEntry(&historyList[i]))
	}

	slices.SortFunc(entries, compareFeedEntries)

	return entries
}

func compareFeedEntries(a, b feedEntry) int {
	if a.CreatedAt.After(b.CreatedAt) {
		return -1
	}
	if a.CreatedAt.Before(b.CreatedAt) {
		return 1
	}

	if a.Kind != b.Kind {
		if a.Kind == feedEntryKindComment {
			return -1
		}
		return 1
	}

	if a.ID > b.ID {
		return -1
	}
	if a.ID < b.ID {
		return 1
	}

	return 0
}

func pageFeedEntries(entries []feedEntry, offset, limit int) []feedEntry {
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	if offset >= len(entries) {
		return make([]feedEntry, 0)
	}

	end := min(offset+limit, len(entries))
	return entries[offset:end]
}
