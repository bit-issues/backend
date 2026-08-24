package projects_test

import (
	"errors"
	"testing"

	"github.com/bit-issues/backend/internal/projects"
)

func TestParseBitbucketRepoURL(t *testing.T) {
	tests := []struct {
		name          string
		repoURL       string
		wantWorkspace string
		wantRepoSlug  string
		wantErr       error
	}{
		// Happy path
		{
			name:          "https url",
			repoURL:       "https://bitbucket.org/workspace/repo-slug",
			wantWorkspace: "workspace",
			wantRepoSlug:  "repo-slug",
		},

		// Trailing slash variants
		{
			name:          "https url with trailing slash",
			repoURL:       "https://bitbucket.org/ws/slug/",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},
		{
			name:          "https url with multiple trailing slashes",
			repoURL:       "https://bitbucket.org/ws/slug///",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},

		// .git suffix
		{
			name:          "https url with .git suffix",
			repoURL:       "https://bitbucket.org/ws/slug.git",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},
		{
			name:          "https url with trailing slash and .git suffix",
			repoURL:       "https://bitbucket.org/ws/slug.git/",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},

		// SSH (scp-like) form
		{
			name:          "ssh scp-like url with .git suffix",
			repoURL:       "git@bitbucket.org:ws/slug.git",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},
		{
			name:          "ssh scp-like url without .git suffix",
			repoURL:       "git@bitbucket.org:ws/slug",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},

		// Case-insensitive host matching
		{name: "mixed case host", repoURL: "https://Bitbucket.org/ws/slug", wantWorkspace: "ws", wantRepoSlug: "slug"},
		{
			name:          "uppercase scheme and host",
			repoURL:       "HTTPS://BITBUCKET.ORG/ws/slug",
			wantWorkspace: "ws",
			wantRepoSlug:  "slug",
		},

		// Allowed charset: alphanumerics, dashes, underscores, dots
		{
			name:          "charset in workspace and slug",
			repoURL:       "https://bitbucket.org/Ws_1/Sub.Repo-2",
			wantWorkspace: "Ws_1",
			wantRepoSlug:  "Sub.Repo-2",
		},

		// Errors
		{name: "non bitbucket host", repoURL: "https://github.com/ws/slug", wantErr: projects.ErrInvalidURL},
		{
			name:    "spoofed subdomain host",
			repoURL: "https://bitbucket.org.evil.com/ws/slug",
			wantErr: projects.ErrInvalidURL,
		},
		{name: "missing slug", repoURL: "https://bitbucket.org/ws", wantErr: projects.ErrInvalidURL},
		{
			name:    "missing slug with trailing slash",
			repoURL: "https://bitbucket.org/ws/",
			wantErr: projects.ErrInvalidURL,
		},
		{name: "extra path segment", repoURL: "https://bitbucket.org/ws/slug/extra", wantErr: projects.ErrInvalidURL},
		{name: "empty string", repoURL: "", wantErr: projects.ErrInvalidURL},
		{name: "blank string", repoURL: "   ", wantErr: projects.ErrInvalidURL},
		{name: "unparseable url", repoURL: ":", wantErr: projects.ErrInvalidURL},
		{name: "no scheme and no colon", repoURL: "bitbucket.org/ws/slug", wantErr: projects.ErrInvalidURL},
		{name: "http scheme rejected", repoURL: "http://bitbucket.org/ws/slug", wantErr: projects.ErrInvalidURL},
		{name: "ftp scheme rejected", repoURL: "ftp://bitbucket.org/ws/slug", wantErr: projects.ErrInvalidURL},
		{
			name:    "invalid character in workspace",
			repoURL: "https://bitbucket.org/w!s/slug",
			wantErr: projects.ErrInvalidURL,
		},
		{name: "invalid character in slug", repoURL: "https://bitbucket.org/ws/slu^g", wantErr: projects.ErrInvalidURL},
		{name: "dot dot workspace", repoURL: "https://bitbucket.org/../etc", wantErr: projects.ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repoSlug, err := projects.ParseBitbucketRepoURL(tt.repoURL)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("projects.ParseBitbucketRepoURL(%q) error = nil, want %v", tt.repoURL, tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("projects.ParseBitbucketRepoURL(%q) error = %v, want %v", tt.repoURL, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("projects.ParseBitbucketRepoURL(%q) unexpected error: %v", tt.repoURL, err)
			}
			if workspace != tt.wantWorkspace {
				t.Errorf(
					"projects.ParseBitbucketRepoURL(%q) workspace = %q, want %q",
					tt.repoURL,
					workspace,
					tt.wantWorkspace,
				)
			}
			if repoSlug != tt.wantRepoSlug {
				t.Errorf(
					"projects.ParseBitbucketRepoURL(%q) repoSlug = %q, want %q",
					tt.repoURL,
					repoSlug,
					tt.wantRepoSlug,
				)
			}
		})
	}
}
