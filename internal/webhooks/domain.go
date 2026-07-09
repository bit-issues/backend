package webhooks

import "strings"

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
