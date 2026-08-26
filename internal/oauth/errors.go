package oauth

import "errors"

var (
	// ErrTokenIssueFailed is returned when the Bitbucket OAuth token
	// endpoint rejects an authorization code or refresh token grant.
	ErrTokenIssueFailed = errors.New("oauth token issue failed")

	// ErrNotFound is returned when a token is not found.
	ErrNotFound = errors.New("token not found")
)
