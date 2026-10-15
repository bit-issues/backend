package commitref

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/bit-issues/backend/internal/tasks"
)

// hashRefPattern matches "#NUMBER" preceded by start-of-string or a non-word character.
var hashRefPattern = regexp.MustCompile(`(?:^|\W)#(\d+)\b`)

// prefixedRefPattern matches "slug#NUMBER" where the slug is preceded by
// start-of-string or a single opening boundary character (whitespace or open
// punctuation). RE2 has no lookbehind, so the boundary character is consumed
// by a non-capturing group and the slug is captured in submatch 1.
var prefixedRefPattern = regexp.MustCompile(`(?i)(?:^|[\s\[("'])([a-z0-9][a-z0-9_-]*?)#(\d+)\b`)

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
		`(?i)\b(` + strings.Join(keys, "|") + `)\s+(?:([a-z0-9][a-z0-9_-]*?)#|#)(\d+)\b`,
	)

	return &KeywordParser{actions: actions, pattern: pattern}, nil
}

// ParseCommitMessage scans a commit message for task references.
// Returns a list of ParsedReference, one per unique (slug, number) pair found.
// A reference may carry an optional project-slug prefix (e.g. "resto-web#13");
// a bare "#NUMBER" keeps ProjectSlug == "" and refers to the current project.
// If both a keyword and bare reference exist for the same (slug, number), the
// keyword wins.
func (kp *KeywordParser) ParseCommitMessage(message string) []ParsedReference {
	seen := make(map[refKey]bool)

	keywordRefs, keywordRanges := kp.parseKeywordRefs(message, seen)
	prefixedRefs, prefixedRanges := parsePrefixedRefs(message, seen, keywordRanges)
	bareRefs := parseBareRefs(message, seen, keywordRanges, prefixedRanges)

	var refs []ParsedReference
	refs = append(refs, keywordRefs...)
	refs = append(refs, prefixedRefs...)
	refs = append(refs, bareRefs...)

	return refs
}

// parseKeywordRefs scans keyword matches and records each unique reference,
// returning the references and the message ranges they occupy.
func (kp *KeywordParser) parseKeywordRefs(message string, seen map[refKey]bool) ([]ParsedReference, []matchRange) {
	keywordMatches := kp.pattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference
	var ranges []matchRange

	for _, m := range keywordMatches {
		keyword := strings.ToLower(message[m[2]:m[3]])

		slug := ""
		if m[4] >= 0 {
			slug = message[m[4]:m[5]]
		}
		number, _ := strconv.Atoi(message[m[6]:m[7]])

		key := refKey{slug: strings.ToLower(slug), number: number}
		if seen[key] {
			continue
		}
		seen[key] = true

		ranges = append(ranges, matchRange{start: m[0], end: m[1]})

		action := kp.actions[keyword]
		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			ProjectSlug:   slug,
			Action:        &KeywordAction{Status: action.Status, Verb: action.Verb},
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs, ranges
}

// refKey identifies a reference by its lowercased project slug and task number.
// Bare references (no slug) use an empty slug.
type refKey struct {
	slug   string
	number int
}

type matchRange struct{ start, end int }

func (r matchRange) overlaps(other matchRange) bool {
	return r.start < other.end && other.start < r.end
}

// overlapsAny reports whether r overlaps any range in ranges.
func overlapsAny(r matchRange, ranges []matchRange) bool {
	return slices.ContainsFunc(ranges, r.overlaps)
}

// parsePrefixedRefs scans slug#NUMBER references not already seen and not
// overlapping a keyword match, returning the references and their ranges.
func parsePrefixedRefs(
	message string,
	seen map[refKey]bool,
	keywordRanges []matchRange,
) ([]ParsedReference, []matchRange) {
	prefixedMatches := prefixedRefPattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference
	var ranges []matchRange

	for _, m := range prefixedMatches {
		slug := message[m[2]:m[3]]
		number, _ := strconv.Atoi(message[m[4]:m[5]])

		key := refKey{slug: strings.ToLower(slug), number: number}
		if seen[key] {
			continue
		}

		pr := matchRange{start: m[0], end: m[1]}
		if overlapsAny(pr, keywordRanges) {
			continue
		}

		seen[key] = true
		ranges = append(ranges, pr)

		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			ProjectSlug:   slug,
			Action:        nil,
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs, ranges
}

// parseBareRefs scans #NUMBER references not already seen and not overlapping a
// keyword or prefixed match.
func parseBareRefs(message string, seen map[refKey]bool, keywordRanges, prefixedRanges []matchRange) []ParsedReference {
	hashMatches := hashRefPattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference

	for _, m := range hashMatches {
		number, _ := strconv.Atoi(message[m[2]:m[3]])

		key := refKey{slug: "", number: number}
		if seen[key] {
			continue
		}

		hr := matchRange{start: m[0], end: m[1]}
		if overlapsAny(hr, keywordRanges) || overlapsAny(hr, prefixedRanges) {
			continue
		}

		seen[key] = true

		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			ProjectSlug:   "",
			Action:        nil,
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs
}

// ParsedReference represents a single task reference found in a commit message.
type ParsedReference struct {
	TaskNumber    int
	ProjectSlug   string         // empty means the current project
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
