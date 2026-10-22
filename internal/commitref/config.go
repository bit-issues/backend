package commitref

// Config holds the settings for parsing commit messages.
type Config struct {
	// ActionKeywords maps a commit message keyword to the task status
	// transition it triggers.
	ActionKeywords map[string]KeywordEntry
}
