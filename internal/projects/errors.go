package projects

import "errors"

var (
	// ErrValidationFailed is returned when input data fails validation.
	ErrValidationFailed = errors.New("validation failed")

	// ErrNotFound is returned when a project with the given ID does not exist.
	ErrNotFound = errors.New("project not found")

	// ErrNameAlreadyUsed is returned when attempting to create or update
	// a project with a name that is already in use by another project.
	ErrNameAlreadyUsed = errors.New("project name already in use")

	// ErrInvalidURL is returned when the repository URL is not in a valid
	// format.
	ErrInvalidURL = errors.New("invalid repository URL")
)
