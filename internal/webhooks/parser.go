package webhooks

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/bit-issues/backend/internal/tasks"
)

// KeywordAction maps a commit message keyword to a task status transition.
type KeywordAction struct {
	Status tasks.Status // target status
	Verb   string       // past-tense label for comment, e.g. "Resolved"
}

// hashRefPattern matches "#NUMBER" preceded by start-of-string or a non-word character.
var hashRefPattern = regexp.MustCompile(`(?:^|\W)#(\d+)\b`)

// KeywordParser holds the configured keyword-to-action mappings and a compiled
// regex for matching them in commit messages.
type KeywordParser struct {
	actions map[string]KeywordAction
	pattern *regexp.Regexp
}

// NewKeywordParser builds a KeywordParser from a config map of keyword entries.
// It validates each status and defaults the Verb to the title-cased keyword
// when empty. The regex is compiled from sorted keys for determinism.
func NewKeywordParser(entries map[string]KeywordEntry) (*KeywordParser, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: action_keywords", ErrEmptyKeyword)
	}

	actions := make(map[string]KeywordAction, len(entries))
	keys := make([]string, 0, len(entries))

	for keyword, entry := range entries {
		if keyword == "" {
			return nil, fmt.Errorf("%w: keyword", ErrEmptyKeyword)
		}
		kw := strings.ToLower(keyword)
		status := tasks.Status(entry.Status)
		if !status.IsValid() {
			return nil, fmt.Errorf("%w: status %q for keyword %q", ErrInvalidStatus, entry.Status, keyword)
		}

		verb := entry.Verb
		if verb == "" {
			verb = titleCase(kw)
		}

		actions[kw] = KeywordAction{Status: status, Verb: verb}
		keys = append(keys, regexp.QuoteMeta(kw))
	}

	sort.Strings(keys)

	pattern := regexp.MustCompile(
		`(?i)\b(` + strings.Join(keys, "|") + `)\s+#(\d+)\b`,
	)

	return &KeywordParser{actions: actions, pattern: pattern}, nil
}

// ParseCommitMessage scans a commit message for task references.
// Returns a list of ParsedReference, one per unique #NUMBER found.
// If both a keyword and bare reference exist for the same number, the keyword wins.
func (kp *KeywordParser) ParseCommitMessage(message string) []ParsedReference {
	keywordMatches := kp.pattern.FindAllStringSubmatchIndex(message, -1)

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

		action := kp.actions[keyword]
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

// titleCase capitalises the first letter of each space-separated word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToTitle(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
