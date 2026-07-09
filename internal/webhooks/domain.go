package webhooks

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bit-issues/backend/internal/tasks"
)

const (
	verbResolved = "Resolved"
	verbClosed   = "Closed"
	verbOnHold   = "On Hold"
)

// keywordActions is the registry of recognized keywords.
// Extend this map to add new auto-transition keywords.
//
//nolint:gochecknoglobals // this is a constant
var keywordActions = map[string]KeywordAction{
	"fix":      {Status: tasks.StatusResolved, Verb: verbResolved},
	"fixes":    {Status: tasks.StatusResolved, Verb: verbResolved},
	"fixed":    {Status: tasks.StatusResolved, Verb: verbResolved},
	"resolve":  {Status: tasks.StatusResolved, Verb: verbResolved},
	"resolves": {Status: tasks.StatusResolved, Verb: verbResolved},
	"resolved": {Status: tasks.StatusResolved, Verb: verbResolved},
	"close":    {Status: tasks.StatusClosed, Verb: verbClosed},
	"closes":   {Status: tasks.StatusClosed, Verb: verbClosed},
	"closed":   {Status: tasks.StatusClosed, Verb: verbClosed},
	"block":    {Status: tasks.StatusOnHold, Verb: verbOnHold},
	"blocks":   {Status: tasks.StatusOnHold, Verb: verbOnHold},
	"blocked":  {Status: tasks.StatusOnHold, Verb: verbOnHold},
	"on hold":  {Status: tasks.StatusOnHold, Verb: verbOnHold},
}

// KeywordAction maps a commit message keyword to a task status transition.
type KeywordAction struct {
	Status tasks.Status // target status
	Verb   string       // past-tense label for comment, e.g. "Resolved"
}

var (
	// keywordRefPattern matches patterns like "fixes #123", "closes #456", etc.
	keywordRefPattern = regexp.MustCompile(
		`(?i)\b(fix|fixes|fixed|resolve|resolves|resolved|close|closes|closed|block|blocks|blocked|on hold)\s+#(\d+)\b`,
	)
	// hashRefPattern matches "#NUMBER" preceded by start-of-string or a non-word character.
	hashRefPattern = regexp.MustCompile(`(?:^|\W)#(\d+)\b`)
)

type matchRange struct{ start, end int }

func (r matchRange) overlaps(other matchRange) bool {
	return r.start < other.end && other.start < r.end
}

// ParsedReference represents a single task reference found in a commit message.
type ParsedReference struct {
	TaskNumber    int
	Action        *KeywordAction // nil if bare #N with no keyword
	CommitHash    string
	CommitMessage string
}

// ParseCommitMessage scans a commit message for task references.
// Returns a list of ParsedReference, one per unique #NUMBER found.
// If both a keyword and bare reference exist for the same number, the keyword wins.
func ParseCommitMessage(message string) []ParsedReference {
	keywordMatches := keywordRefPattern.FindAllStringSubmatchIndex(message, -1)

	var keywordRanges []matchRange
	var refs []ParsedReference
	seen := make(map[int]bool)

	for _, m := range keywordMatches {
		keyword := strings.ToLower(message[m[2]:m[3]])
		numStr := message[m[4]:m[5]]
		number, _ := strconv.Atoi(numStr)

		if seen[number] {
			continue
		}
		seen[number] = true

		keywordRanges = append(keywordRanges, matchRange{start: m[0], end: m[1]})

		action := keywordActions[keyword]
		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			Action:        &KeywordAction{Status: action.Status, Verb: action.Verb},
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	// Find #NUMBER references not part of keyword matches
	hashMatches := hashRefPattern.FindAllStringSubmatchIndex(message, -1)
	for _, m := range hashMatches {
		numStr := message[m[2]:m[3]]
		number, _ := strconv.Atoi(numStr)

		if seen[number] {
			continue
		}
		seen[number] = true

		hr := matchRange{start: m[0], end: m[1]}
		consumed := false
		for _, kr := range keywordRanges {
			if kr.overlaps(hr) {
				consumed = true
				break
			}
		}
		if consumed {
			continue
		}

		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			Action:        nil,
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs
}

// PushCommit is a flattened commit used by the service layer.
type PushCommit struct {
	Hash    string
	Message string
	Author  string
}

// ProcessResult tracks the results of processing a push event.
type ProcessResult struct {
	Matched   int `json:"matched"`
	Resolved  int `json:"resolved"`
	Mentioned int `json:"mentioned"`
}

func NewProcessResult() *ProcessResult {
	return &ProcessResult{}
}

func (r *ProcessResult) AddMatched() {
	r.Matched++
}

func (r *ProcessResult) AddResolved() {
	r.Resolved++
	r.Matched++
}

func (r *ProcessResult) AddMentioned() {
	r.Mentioned++
	r.Matched++
}

// firstLine returns the first line of a multi-line string, trimmed.
func firstLine(msg string) string {
	line, _, _ := strings.Cut(msg, "\n")
	return strings.TrimSpace(line)
}
