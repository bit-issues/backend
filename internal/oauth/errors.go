package oauth

import "errors"

var (
	// ErrDisabled is returned when the OAuth service is disabled.
	ErrDisabled = errors.New("oauth service disabled")

	// ErrTokenIssueFailed is returned when the Bitbucket OAuth token
	// endpoint rejects an authorization code or refresh token grant.
	ErrTokenIssueFailed = errors.New("oauth token issue failed")

	// ErrNotFound is returned when a token is not found.
	ErrNotFound = errors.New("token not found")

	// ErrStateNotFound is returned when an OAuth CSRF state is missing,
	// expired, or already consumed.
	ErrStateNotFound = errors.New("oauth state not found")
)
