package webhooks

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrEmptyKeyword     = errors.New("must not be empty")
	ErrInvalidStatus    = errors.New("invalid keyword status")

	// ErrWebhookSecretNotConfigured is returned when webhook registration is
	// attempted without a configured WEBHOOKS__SECRET.
	ErrWebhookSecretNotConfigured = errors.New("webhook secret not configured")

	// ErrBitbucketNotConfigured is returned when webhook operations are
	// attempted without any Bitbucket credential: neither an OAuth token nor
	// a static BITBUCKET__ACCESS_TOKEN.
	ErrBitbucketNotConfigured = errors.New("bitbucket is not configured")
)
