package commitref

import "slices"

// parser extracts task references from a commit message. Parsers run as an
// ordered chain inside Service; each parser claims its matches through
// parseState, which centralizes dedup and overlap precedence across the chain.
// Precedence is chain order: a later parser never shadows an earlier one.
type parser interface {
	parse(message string, st *parseState) []ParsedReference
}

// parseState carries the shared dedup and precedence state for a single
// ParseCommitMessage call across the parser chain.
type parseState struct {
	seen   map[refKey]bool
	ranges []matchRange
}

// claim records the match for key unless the key was already claimed or the
// match overlaps a range claimed by an earlier parser. A failed claim leaves
// the state untouched: the key stays available to later parsers, mirroring the
// original per-parser skip semantics.
func (st *parseState) claim(key refKey, r matchRange) bool {
	if st.seen[key] {
		return false
	}
	if overlapsAny(r, st.ranges) {
		return false
	}

	st.seen[key] = true
	st.ranges = append(st.ranges, r)
	return true
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
