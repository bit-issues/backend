package comments

import "errors"

// Module-specific error definitions.
// These errors can be checked using [errors.Is](err, ErrXXX).
var (
	// ErrNotFound indicates the requested comment does not exist.
	ErrNotFound = errors.New("comment not found")

	// ErrValidationFailed indicates input validation failed.
	ErrValidationFailed = errors.New("validation failed")

	// ErrUnauthorized indicates the user lacks permission for this action.
	ErrUnauthorized = errors.New("unauthorized")
)
