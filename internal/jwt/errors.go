package jwt

import "errors"

var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token has expired")
)
