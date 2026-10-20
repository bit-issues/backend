package bitbucket

import (
	"fmt"
	"net/url"
	"strings"
)

// minRepoPathSegments is the minimum path segment count of a valid
// repository URL (workspace and slug).
const minRepoPathSegments = 2

// ParseRepoSlug extracts the workspace and repository slug from a Bitbucket
// Cloud repository URL of the form https://bitbucket.org/{workspace}/{slug}.
// Trailing slashes, deeper paths, and a .git suffix are tolerated. Non-HTTPS
// URLs, foreign hosts, and paths with fewer than two segments are rejected
// with ErrInvalidRepoURL.
func ParseRepoSlug(repoURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	if u.Scheme != "https" || !strings.EqualFold(u.Host, "bitbucket.org") {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < minRepoPathSegments {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	workspace := segments[0]
	slug := strings.TrimSuffix(segments[1], ".git")
	if workspace == "" || slug == "" {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	return workspace, slug, nil
}
