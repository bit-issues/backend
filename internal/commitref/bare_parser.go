package commitref

import (
	"regexp"
	"strconv"
)

// hashRefPattern matches "#NUMBER" preceded by start-of-string or a non-word,
// non-slash character. The slash exclusion keeps URL fragments like
// "https://x/#13" from being parsed as task references; "/" is a \W character,
// so a plain (?:^|\W) boundary would consume it.
var hashRefPattern = regexp.MustCompile(`(?:^|[^\w/])#(\d+)\b`)

// bareParser matches #NUMBER references referring to the current project; the
// resulting references carry no status transition.
type bareParser struct{}

func (bareParser) parse(message string, st *parseState) []ParsedReference {
	matches := hashRefPattern.FindAllStringSubmatchIndex(message, -1)

	var refs []ParsedReference

	for _, m := range matches {
		number, _ := strconv.Atoi(message[m[2]:m[3]])

		key := refKey{slug: "", number: number}
		if !st.claim(key, matchRange{start: m[0], end: m[1]}) {
			continue
		}

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
