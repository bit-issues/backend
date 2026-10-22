package commitref

// Service parses commit messages for task references by running an ordered
// chain of parsers over shared per-message state.
type Service struct {
	parsers []parser
}

// NewService builds a Service from cfg. It validates every keyword status,
// rejects keywords that collide after case normalization, and compiles the
// keyword matcher from the sorted keyword list. An invalid configuration is
// returned as an error so the app fails at startup instead of silently
// ignoring commit references. An empty ActionKeywords map leaves the keyword
// matcher uncompiled: the chain then runs in mentions-only mode, so bare #N
// and slug#N references still parse but no keyword carries a status
// transition.
func NewService(cfg Config) (*Service, error) {
	kwParser, err := newKeywordParser(cfg)
	if err != nil {
		return nil, err
	}

	return &Service{parsers: []parser{kwParser, prefixedParser{}, bareParser{}}}, nil
}

// ParseCommitMessage scans a commit message for task references.
// Returns a list of ParsedReference, one per unique (slug, number) pair found.
// A reference may carry an optional project-slug prefix (e.g. "resto-web#13");
// a bare "#NUMBER" keeps ProjectSlug == "" and refers to the current project.
// Parsers run in precedence order (keyword, prefixed, bare); if both a keyword
// and bare reference exist for the same (slug, number), the keyword wins.
func (s *Service) ParseCommitMessage(message string) []ParsedReference {
	st := &parseState{seen: make(map[refKey]bool), ranges: nil}

	var refs []ParsedReference
	for _, p := range s.parsers {
		refs = append(refs, p.parse(message, st)...)
	}

	return refs
}
