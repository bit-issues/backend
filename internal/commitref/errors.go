package commitref

import "errors"

var (
	ErrEmptyKeyword  = errors.New("must not be empty")
	ErrInvalidStatus = errors.New("invalid keyword status")
)
