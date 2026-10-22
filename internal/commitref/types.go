package commitref

import "github.com/bit-issues/backend/internal/tasks"

type KeywordEntry struct {
	Status string `json:"status"`
	Verb   string `json:"verb"`
}

// KeywordAction maps a commit message keyword to a task status transition.
type KeywordAction struct {
	Status tasks.Status // target status
	Verb   string       // past-tense label for comment, e.g. "Resolved"
}

// ParsedReference represents a single task reference found in a commit message.
type ParsedReference struct {
	TaskNumber    int
	ProjectSlug   string         // empty means the current project
	Action        *KeywordAction // nil if bare #N with no keyword
	CommitHash    string
	CommitMessage string
}
