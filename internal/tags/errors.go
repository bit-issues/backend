package tags

import "errors"

var (
	ErrNotFound         = errors.New("tag not found")
	ErrValidationFailed = errors.New("validation failed")
)
