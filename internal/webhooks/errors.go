package webhooks

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrEmptyKeyword     = errors.New("must not be empty")
	ErrInvalidStatus    = errors.New("invalid keyword status")

	// ErrWebhookSecretNotConfigured is returned when webhook registration is
	// attempted without a configured WEBHOOKS__SECRET.
	ErrWebhookSecretNotConfigured = errors.New("webhook secret not configured")
)
