package projects

import (
	"fmt"
	"net/url"
	"strings"
)

// bitbucketHost is the canonical Bitbucket host. Host matching is
// case-insensitive.
const bitbucketHost = "bitbucket.org"

// ParseBitbucketRepoURL extracts the workspace and repository slug from a
// Bitbucket repository URL.
//
// Accepted forms (HTTPS and SSH only; other schemes such as HTTP are rejected
// because Bitbucket serves repositories over HTTPS and git+SSH):
//
//   - https://bitbucket.org/{workspace}/{repo_slug}
//   - https://bitbucket.org/{workspace}/{repo_slug}.git
//   - git@bitbucket.org:{workspace}/{repo_slug}[.git]
//
// Trailing slashes are ignored, the ".git" suffix is stripped from the slug,
// and host matching is case-insensitive. The returned workspace and slug are
// not lowercased: Bitbucket slugs are case-preserving.
func ParseBitbucketRepoURL(repoURL string) (string, string, error) {
	if repoURL == "" {
		return "", "", fmt.Errorf("%w: repository URL is required", ErrInvalidURL)
	}

	path, err := repoPath(repoURL)
	if err != nil {
		return "", "", err
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%w: repository URL must contain a workspace and a repository slug", ErrInvalidURL)
	}

	workspace, repoSlug := parts[0], strings.TrimSuffix(parts[1], ".git")
	if !isValidRepoSegment(workspace) || !isValidRepoSegment(repoSlug) {
		return "", "", fmt.Errorf(
			"%w: workspace and repository slug may only contain alphanumeric characters, dashes, underscores and dots",
			ErrInvalidURL,
		)
	}

	return workspace, repoSlug, nil
}

// repoPath extracts the path portion of a Bitbucket repository URL.
func repoPath(repoURL string) (string, error) {
	if strings.Contains(repoURL, "://") {
		return httpsRepoPath(repoURL)
	}

	return sshRepoPath(repoURL)
}

// httpsRepoPath validates an HTTPS repository URL and returns its path.
func httpsRepoPath(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("%w: failed to parse repository URL: %w", ErrInvalidURL, err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("%w: repository URL must use the HTTPS scheme", ErrInvalidURL)
	}
	if !strings.EqualFold(u.Host, bitbucketHost) {
		return "", fmt.Errorf("%w: repository URL must point to %s", ErrInvalidURL, bitbucketHost)
	}

	return u.Path, nil
}

// sshRepoPath validates an SCP-like SSH repository URL and returns its path.
func sshRepoPath(repoURL string) (string, error) {
	// SCP-like SSH syntax: [user@]bitbucket.org:{workspace}/{repo_slug}
	host, path, ok := strings.Cut(repoURL, ":")
	if !ok {
		return "", fmt.Errorf("%w: repository URL is not a valid Bitbucket URL", ErrInvalidURL)
	}
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	if !strings.EqualFold(host, bitbucketHost) {
		return "", fmt.Errorf("%w: repository URL must point to %s", ErrInvalidURL, bitbucketHost)
	}

	return path, nil
}

// isValidRepoSegment reports whether s is a valid Bitbucket workspace or
// repository slug: composed only of ASCII alphanumerics, dashes, underscores
// and dots, and not a bare "." or ".." segment.
func isValidRepoSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
