package webhooks

import "errors"

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrEmptyKeyword     = errors.New("must not be empty")
	ErrInvalidStatus    = errors.New("invalid keyword status")
)
