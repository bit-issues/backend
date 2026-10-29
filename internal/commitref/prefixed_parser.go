package commitref

import (
	"regexp"
	"strconv"
	"strings"
)

// prefixedRefPattern matches "slug#NUMBER" where the slug is preceded by
// start-of-string or a single opening boundary character (whitespace, open
// punctuation, or the ":" and "," delimiters used in commit subjects such as
// "chore:resto-web#13"). Slashes are deliberately not boundaries, so URL paths
// ("https://x/foo#1") do not match. RE2 has no lookbehind, so the boundary
// character is consumed by a non-capturing group and the slug is captured in
// submatch 1.
var prefixedRefPattern = regexp.MustCompile(`(?i)(?:^|[\s\[("':,])([a-z0-9][a-z0-9_-]*?)#(\d+)\b`)

// prefixedParser matches slug#NUMBER references without keyword context; the
// resulting references carry no status transition.
type prefixedParser struct{}

func (prefixedParser) parse(message string, st *parseState) []ParsedReference {
	matches := prefixedRefPattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference

	for _, m := range matches {
		slug := message[m[2]:m[3]]
		number, _ := strconv.Atoi(message[m[4]:m[5]])

		key := refKey{slug: strings.ToLower(slug), number: number}
		if !st.claim(key, matchRange{start: m[0], end: m[1]}) {
			continue
		}

		refs = append(refs, ParsedReference{
			TaskNumber:    number,
			ProjectSlug:   slug,
			Action:        nil,
			CommitHash:    "",
			CommitMessage: message,
		})
	}

	return refs
}
