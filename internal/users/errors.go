package users

import "errors"

var (
	ErrNotFound          = errors.New("user not found")
	ErrEmailAlreadyUsed  = errors.New("email already used")
	ErrInvalidCredential = errors.New("invalid credentials")
	ErrNotActive         = errors.New("user is not active")
)
