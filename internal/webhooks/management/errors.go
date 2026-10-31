package management

import "errors"

var (
	// ErrCallbackURLNotConfigured is returned when BITBUCKET__CALLBACK_URL is
	// empty and webhook management requires it.
	ErrCallbackURLNotConfigured = errors.New("bitbucket callback url not configured")

	// ErrCallbackURLInvalid is returned when the callback URL is not https or
	// carries a query string or fragment.
	ErrCallbackURLInvalid = errors.New("bitbucket callback url invalid")

	// ErrSecretNotConfigured is returned when WEBHOOKS__SECRET is empty and
	// registration requires a verifiable secret.
	ErrSecretNotConfigured = errors.New("webhook secret not configured")

	// ErrOAuthDisabled is returned when the Bitbucket OAuth service is
	// disabled by configuration.
	ErrOAuthDisabled = errors.New("bitbucket oauth disabled")

	// ErrOAuthNotConnected is returned when no usable OAuth token exists.
	ErrOAuthNotConnected = errors.New("bitbucket oauth not connected")

	// ErrNotBitbucketRepo is returned when a project repository URL is not a
	// bitbucket.org URL.
	ErrNotBitbucketRepo = errors.New("not a bitbucket repository")
)
