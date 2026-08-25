package oauth

import "errors"

var (
	// ErrOAuthNotConnected is returned when no OAuth token row exists.
	ErrOAuthNotConnected = errors.New("oauth not connected")

	// ErrStateNotFound is returned when a CSRF state is unknown, expired,
	// already consumed, or bound to a different user or redirect URI.
	ErrStateNotFound = errors.New("oauth state not found or expired")

	// ErrInvalidScope is returned when the OAuth scope set lacks 'webhook'.
	ErrInvalidScope = errors.New("invalid oauth scope: 'webhook' required")

	// ErrRefreshNotConfigured is returned when no token refresher is wired.
	ErrRefreshNotConfigured = errors.New("oauth token refresh is not configured")

	// ErrTokenExpired is returned when a token cannot be refreshed and its
	// lifetime has ended (e.g. no refresh token stored).
	ErrTokenExpired = errors.New("oauth token expired and cannot be refreshed")

	// ErrTokenExchangeFailed is returned when the Bitbucket OAuth token
	// endpoint rejects an authorization code or refresh token grant.
	ErrTokenExchangeFailed = errors.New("oauth token exchange failed")

	// ErrOAuthRevoked is returned when a stored OAuth credential exists but
	// is revoked, expired, or cannot be refreshed. Callers must surface it
	// and must never fall back to the static token.
	ErrOAuthRevoked = errors.New("oauth token revoked or invalid")

	// errUnexpectedTokenType guards the singleflight result type assertion.
	errUnexpectedTokenType = errors.New("unexpected oauth token type")
)
