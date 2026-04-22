package attachments

import "errors"

var (
	ErrNotFound         = errors.New("attachment not found")
	ErrValidationFailed = errors.New("validation failed")
	ErrTaskNotFound     = errors.New("task not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrFileTooLarge     = errors.New("file exceeds maximum allowed size")
	ErrNotUploaded      = errors.New("attachment is not yet uploaded")
)
