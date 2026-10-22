package commitref

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/bit-issues/backend/internal/tasks"
)

// keywordParser matches configured action keywords and attaches status
// transitions to the references they carry.
type keywordParser struct {
	actions map[string]KeywordAction
	pattern *regexp.Regexp
}

// newKeywordParser validates cfg.ActionKeywords and compiles the keyword
// matcher from the sorted keyword list. An empty map yields a parser with no
// matcher that claims nothing, so the chain runs in mentions-only mode; an
// invalid configuration is returned as an error so the app fails at startup
// instead of silently ignoring commit references.
func newKeywordParser(cfg Config) (*keywordParser, error) {
	actions := make(map[string]KeywordAction, len(cfg.ActionKeywords))
	keys := make([]string, 0, len(cfg.ActionKeywords))

	for keyword, entry := range cfg.ActionKeywords {
		if keyword == "" {
			return nil, fmt.Errorf("%w: keyword", ErrEmptyKeyword)
		}
		kw := strings.ToLower(keyword)
		if _, exists := actions[kw]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateKeyword, kw)
		}

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

	if len(keys) == 0 {
		return &keywordParser{actions: actions, pattern: nil}, nil
	}

	sort.Strings(keys)

	return &keywordParser{
		actions: actions,
		pattern: regexp.MustCompile(
			`(?i)\b(` + strings.Join(keys, "|") + `)\s+(?:([a-z0-9][a-z0-9_-]*?)#|#)(\d+)\b`,
		),
	}, nil
}

// parse scans keyword matches, claiming each unique (slug, number) pair. A
// claimed reference carries the keyword's status transition; the keyword wins
// over later parsers for the same pair because it runs first. A parser built
// from an empty keyword map has no matcher and claims nothing.
func (p *keywordParser) parse(message string, st *parseState) []ParsedReference {
	if p.pattern == nil {
		return nil
	}

	matches := p.pattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference

	for _, m := range matches {
		keyword := strings.ToLower(message[m[2]:m[3]])

		// The (?i) flag folds some characters that strings.ToLower leaves
		// unchanged (e.g. U+017F long s), so a match may not have a configured
		// action. Skip it before claiming, leaving the pair to later parsers.
		action, ok := p.actions[keyword]
		if !ok {
			continue
		}

		slug := ""
		if m[4] >= 0 {
			slug = message[m[4]:m[5]]
		}
		number, _ := strconv.Atoi(message[m[6]:m[7]])

		key := refKey{slug: strings.ToLower(slug), number: number}
		if !st.claim(key, matchRange{start: m[0], end: m[1]}) {
			continue
		}

		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			ProjectSlug:   slug,
			Action:        &KeywordAction{Status: action.Status, Verb: action.Verb},
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs
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
