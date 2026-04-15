package tasks

import "errors"

// Module-specific error definitions.
// These errors can be checked using [errors.Is](err, ErrXXX).
var (
	// ErrNotFound indicates the requested task does not exist.
	ErrNotFound = errors.New("task not found")

	// ErrValidationFailed indicates input validation failed.
	ErrValidationFailed = errors.New("validation failed")

	// ErrProjectNotFound indicates the specified project does not exist.
	ErrProjectNotFound = errors.New("project not found")
)
