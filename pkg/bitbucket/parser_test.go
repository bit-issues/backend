package bitbucket_test

import (
	"errors"
	"testing"

	"github.com/bit-issues/backend/pkg/bitbucket"
)

func TestParseRepoSlug(t *testing.T) {
	tests := []struct {
		name string
		url  string
		ws   string
		slug string
		err  bool
	}{
		{"simple", "https://bitbucket.org/acme/widgets", "acme", "widgets", false},
		{"trailing slash", "https://bitbucket.org/acme/widgets/", "acme", "widgets", false},
		{"git suffix", "https://bitbucket.org/acme/widgets.git", "acme", "widgets", false},
		{"git suffix and trailing slash", "https://bitbucket.org/acme/widgets.git/", "acme", "widgets", false},
		{"deep path", "https://bitbucket.org/acme/widgets/src/main/README.md", "acme", "widgets", false},
		{"uppercase host", "https://BITBUCKET.ORG/acme/widgets", "acme", "widgets", false},
		{"whitespace around", "  https://bitbucket.org/acme/widgets  ", "acme", "widgets", false},
		{"empty", "", "", "", true},
		{"non-https", "http://bitbucket.org/acme/widgets", "", "", true},
		{"foreign host", "https://github.com/acme/widgets", "", "", true},
		{"missing slug", "https://bitbucket.org/acme", "", "", true},
		{"missing path", "https://bitbucket.org", "", "", true},
		{"trailing slash only", "https://bitbucket.org/", "", "", true},
		{"double slash", "https://bitbucket.org//", "", "", true},
		{"empty segments", "https://bitbucket.org//widgets", "", "", true},
		{"garbage", "::not a url::", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, slug, err := bitbucket.ParseRepoSlug(tt.url)
			if tt.err {
				if !errors.Is(err, bitbucket.ErrInvalidRepoURL) {
					t.Fatalf("expected ErrInvalidRepoURL, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ws != tt.ws || slug != tt.slug {
				t.Fatalf("got (%q, %q), want (%q, %q)", ws, slug, tt.ws, tt.slug)
			}
		})
	}
}
