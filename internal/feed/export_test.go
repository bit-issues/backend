package feed

// DeriveHistoryChangesForTest exposes deriveHistoryChanges to the external test package.
var DeriveHistoryChangesForTest = deriveHistoryChanges

// DeriveChangesForTest exposes deriveChanges to the external test package.
var DeriveChangesForTest = deriveChanges

// ResolveChangeValuesForTest exposes resolveChangeValues to the external test package.
var ResolveChangeValuesForTest = resolveChangeValues

// FeedWindowForTest exposes feedWindow to the external test package.
func FeedWindowForTest(offset, limit int) (int, bool) {
	return feedWindow(offset, limit)
}
