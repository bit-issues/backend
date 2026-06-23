package webauthn

import "errors"

var (
	ErrCredentialNotFound     = errors.New("credential not found")
	ErrSessionNotFound        = errors.New("session not found")
	ErrDuplicateCredential    = errors.New("credential already registered")
	ErrInvalidWebAuthnPayload = errors.New("invalid webauthn request payload")

	ErrUnexpectedType = errors.New("unexpected type")
)
