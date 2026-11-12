package tasks

// DeriveHistoryChangesForTest exposes deriveHistoryChanges to the external test package.
var DeriveHistoryChangesForTest = deriveHistoryChanges

// FeedWindowForTest exposes feedWindow to the external test package.
func FeedWindowForTest(offset, limit int) (int, bool) {
	return feedWindow(offset, limit)
}
