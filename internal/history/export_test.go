package history

// Entry builders expose the unexported entry constructors to the external
// test package.
var (
	NewCreatedEntryForTest           = newCreatedEntry
	NewUpdatedEntryForTest           = newUpdatedEntry
	NewAttachmentAddedEntryForTest   = newAttachmentAddedEntry
	NewAttachmentRemovedEntryForTest = newAttachmentRemovedEntry
)
