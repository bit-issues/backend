package webhooks_test

import (
	"reflect"
	"testing"

	"github.com/bit-issues/backend/internal/webhooks"
)

// The webhook Service depends on concrete services backed by a *bun.DB, and the
// repository offers no in-memory/unmocked DB harness (MySQL via bun; no SQLite
// driver in deps). Constructing a Service for an end-to-end
// TestProcessCommit_CrossProjectRef would require either a new test dependency
// or a testability seam on the Service, both disallowed by the plan and owner
// conventions. These tests therefore cover the pure resolution/dedup helpers
// that processCommit uses. What remains verified by inspection/manually:
//   - projects.GetBySlug resolution of a lowercased slug (service.go processCommit)
//   - log-and-skip on unknown slug without aborting remaining refs
//   - task lookup, status transition, and bot comment for the resolved project

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already lowercase", "resto-web", "resto-web"},
		{"mixed case", "Resto-Web", "resto-web"},
		{"all upper", "API", "api"},
		{"with digits", "Fix-2FA", "fix-2fa"},
		{"empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := webhooks.NormalizeSlugForTest(tc.in); got != tc.want {
				t.Fatalf("NormalizeSlugForTest(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDedupResolvedRefs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []webhooks.ResolvedRefView
		want []webhooks.ResolvedRefView
	}{
		{
			name: "same project and number collapse",
			in: []webhooks.ResolvedRefView{
				{ProjectID: "myproj", TaskNumber: 13},
				{ProjectID: "myproj", TaskNumber: 13},
			},
			want: []webhooks.ResolvedRefView{
				{ProjectID: "myproj", TaskNumber: 13},
			},
		},
		{
			name: "different projects same number kept",
			in: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13},
				{ProjectID: "api", TaskNumber: 13},
			},
			want: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13},
				{ProjectID: "api", TaskNumber: 13},
			},
		},
		{
			name: "same project different numbers kept",
			in: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13},
				{ProjectID: "resto-web", TaskNumber: 7},
			},
			want: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13},
				{ProjectID: "resto-web", TaskNumber: 7},
			},
		},
		{
			name: "first occurrence wins and order preserved",
			in: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13, CommitHash: "aaaaaaa"},
				{ProjectID: "api", TaskNumber: 7, CommitHash: "bbbbbbb"},
				{ProjectID: "resto-web", TaskNumber: 13, CommitHash: "ccccccc"},
			},
			want: []webhooks.ResolvedRefView{
				{ProjectID: "resto-web", TaskNumber: 13, CommitHash: "aaaaaaa"},
				{ProjectID: "api", TaskNumber: 7, CommitHash: "bbbbbbb"},
			},
		},
		{
			name: "empty input",
			in:   nil,
			want: []webhooks.ResolvedRefView{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := webhooks.DedupResolvedRefsForTest(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("DedupResolvedRefsForTest(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}
