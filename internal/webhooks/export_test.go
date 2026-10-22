package webhooks

import "github.com/bit-issues/backend/internal/commitref"

// This file is compiled only during testing. It exposes the unexported
// resolution/dedup helpers to the external test package without expanding the
// production API (owner convention: white-box access via export_test.go).

// NormalizeSlugForTest exposes normalizeSlug to the external test package.
func NormalizeSlugForTest(slug string) string {
	return normalizeSlug(slug)
}

// ResolvedRefView is a test-only projection of a service-resolved reference.
type ResolvedRefView struct {
	ProjectID  string
	TaskNumber int
	CommitHash string
}

// DedupResolvedRefsForTest exposes dedupResolvedRefs to the external test
// package using a comparable projection.
func DedupResolvedRefsForTest(refs []ResolvedRefView) []ResolvedRefView {
	in := make([]resolvedRef, 0, len(refs))
	for _, r := range refs {
		in = append(in, resolvedRef{
			projectID: r.ProjectID,
			ref:       commitref.ParsedReference{TaskNumber: r.TaskNumber, CommitHash: r.CommitHash},
		})
	}

	out := dedupResolvedRefs(in)
	views := make([]ResolvedRefView, 0, len(out))
	for _, r := range out {
		views = append(views, ResolvedRefView{
			ProjectID:  r.projectID,
			TaskNumber: r.ref.TaskNumber,
			CommitHash: r.ref.CommitHash,
		})
	}

	return views
}
