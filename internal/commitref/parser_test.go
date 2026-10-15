package commitref_test

import (
	"testing"

	"github.com/bit-issues/backend/internal/commitref"
)

func testParser(t *testing.T) *commitref.KeywordParser {
	t.Helper()
	kp, err := commitref.NewKeywordParser(map[string]commitref.KeywordEntry{
		"fixes":    {Status: "Resolved", Verb: "Resolved"},
		"closes":   {Status: "Closed", Verb: "Closed"},
		"resolves": {Status: "Resolved", Verb: "Resolved"},
	})
	if err != nil {
		t.Fatalf("failed to build parser: %v", err)
	}
	return kp
}

// refView is a comparable projection of ParsedReference for table assertions.
type refView struct {
	slug    string
	number  int
	hasVerb bool
	verb    string
	status  string
}

func toViews(refs []commitref.ParsedReference) []refView {
	views := make([]refView, 0, len(refs))
	for _, r := range refs {
		v := refView{slug: r.ProjectSlug, number: r.TaskNumber}
		if r.Action != nil {
			v.hasVerb = true
			v.verb = r.Action.Verb
			v.status = string(r.Action.Status)
		}
		views = append(views, v)
	}
	return views
}

func equalViews(a, b []refView) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseCommitMessage_Table(t *testing.T) {
	kp := testParser(t)

	cases := []struct {
		name string
		msg  string
		want []refView
	}{
		{
			name: "bare prefixed",
			msg:  "resto-web#13",
			want: []refView{{slug: "resto-web", number: 13}},
		},
		{
			name: "keyword prefixed",
			msg:  "fixes resto-web#13",
			want: []refView{{slug: "resto-web", number: 13, hasVerb: true, verb: "Resolved", status: "Resolved"}},
		},
		{
			name: "keyword prefixed at start with surrounding text",
			msg:  "chore: fixes resto-web#13 now",
			want: []refView{{slug: "resto-web", number: 13, hasVerb: true, verb: "Resolved", status: "Resolved"}},
		},
		{
			name: "case-insensitive slug captured raw",
			msg:  "Resto-Web#13",
			want: []refView{{slug: "Resto-Web", number: 13}},
		},
		{
			name: "compat bare",
			msg:  "#13",
			want: []refView{{slug: "", number: 13}},
		},
		{
			name: "compat bare embedded",
			msg:  "some work refs #42 done",
			want: []refView{{slug: "", number: 42}},
		},
		{
			name: "compat keyword bare",
			msg:  "fixes #13",
			want: []refView{{slug: "", number: 13, hasVerb: true, verb: "Resolved", status: "Resolved"}},
		},
		{
			name: "multiple refs prefixed and bare",
			msg:  "fixes resto-web#13 and api#7 then #42",
			want: []refView{
				{slug: "resto-web", number: 13, hasVerb: true, verb: "Resolved", status: "Resolved"},
				{slug: "api", number: 7},
				{slug: "", number: 42},
			},
		},
		{
			name: "dedup prefixed and bare keep distinct slugs",
			msg:  "myproj#13 and #13",
			want: []refView{
				{slug: "myproj", number: 13},
				{slug: "", number: 13},
			},
		},
		{
			name: "dedup identical case-insensitive prefixed",
			msg:  "MyProj#13 and myproj#13",
			want: []refView{{slug: "MyProj", number: 13}},
		},
		{
			name: "keyword wins over prefixed bare same pair",
			msg:  "fixes resto-web#13 then resto-web#13 again",
			want: []refView{{slug: "resto-web", number: 13, hasVerb: true, verb: "Resolved", status: "Resolved"}},
		},
		{
			name: "negative email-like context",
			msg:  "a@b#1",
			want: nil,
		},
		{
			name: "negative url fragment",
			msg:  "https://x/page#1",
			want: nil,
		},
		{
			name: "negative hash without digits",
			msg:  "resto-web#",
			want: nil,
		},
		{
			name: "boundary open bracket prefixed",
			msg:  "see [resto-web#13] please",
			want: []refView{{slug: "resto-web", number: 13}},
		},
		{
			name: "boundary parenthesis prefixed",
			msg:  "see (resto-web#13) please",
			want: []refView{{slug: "resto-web", number: 13}},
		},
		{
			name: "slug with hyphen and digits",
			msg:  "fix-2fa#9",
			want: []refView{{slug: "fix-2fa", number: 9}},
		},
		{
			name: "empty message",
			msg:  "",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toViews(kp.ParseCommitMessage(tc.msg))
			if !equalViews(got, tc.want) {
				t.Fatalf("ParseCommitMessage(%q):\n got %+v\nwant %+v", tc.msg, got, tc.want)
			}
		})
	}
}

// TestParseCommitMessage_FieldRouting verifies CommitHash/CommitMessage routing
// and that Action is nil for bare references and non-nil for keyword references.
func TestParseCommitMessage_FieldRouting(t *testing.T) {
	kp := testParser(t)

	refs := kp.ParseCommitMessage("resto-web#13")
	if len(refs) != 1 {
		t.Fatalf("want 1 ref, got %d", len(refs))
	}
	if refs[0].Action != nil {
		t.Fatalf("bare prefixed ref should have nil Action, got %+v", refs[0].Action)
	}
	if refs[0].CommitMessage != "resto-web#13" {
		t.Fatalf("CommitMessage not routed: %q", refs[0].CommitMessage)
	}
	if refs[0].CommitHash != "" {
		t.Fatalf("CommitHash should be empty from parser: %q", refs[0].CommitHash)
	}

	refs = kp.ParseCommitMessage("fixes #13")
	if len(refs) != 1 || refs[0].Action == nil {
		t.Fatalf("keyword ref should have non-nil Action, got %+v", refs)
	}
}
