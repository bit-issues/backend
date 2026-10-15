package webhooks

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
)
