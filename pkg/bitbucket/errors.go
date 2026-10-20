package bitbucket

import (
	"errors"
	"fmt"
)

var (
	// ErrUnauthorized is returned when Bitbucket rejects the OAuth token (401).
	ErrUnauthorized = errors.New("bitbucket unauthorized")

	// ErrForbidden is returned when the token lacks the required permission (403).
	ErrForbidden = errors.New("bitbucket forbidden")

	// ErrNotFound is returned when the Bitbucket resource does not exist (404).
	ErrNotFound = errors.New("bitbucket not found")

	// ErrInvalidRepoURL is returned when a repository URL is not a valid
	// https://bitbucket.org/{workspace}/{slug} URL.
	ErrInvalidRepoURL = errors.New("invalid bitbucket repository URL")
)

// APIError represents an unexpected Bitbucket API response or a transport
// failure. StatusCode is 0 for transport/network failures. Error text never
// includes upstream response bodies, secrets, or the Authorization header.
type APIError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface with a sanitized message.
func (e *APIError) Error() string {
	if e.StatusCode == 0 {
		return "bitbucket api unavailable"
	}
	return fmt.Sprintf("bitbucket api error: status code %d", e.StatusCode)
}
