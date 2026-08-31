package oauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidKey indicates the configured OAuth token encryption key is missing
// or has an unsupported length.
var ErrInvalidKey = errors.New("invalid oauth token encryption key")

// ErrInvalidCiphertext indicates stored token ciphertext could not be
// authenticated or decoded.
var ErrInvalidCiphertext = errors.New("invalid oauth token ciphertext")

// Encryptor performs authenticated encryption (AES-GCM) of OAuth tokens at
// rest. A random 12-byte nonce is prepended to the sealed payload; the
// combined bytes are base64-encoded for storage in a text column. Decryption
// fails closed (returns an error) when the ciphertext is tampered with.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor builds an Encryptor from a raw AES key. The key must be 16, 24,
// or 32 bytes (AES-128/192/256).
func NewEncryptor(key []byte) (*Encryptor, error) {
	if l := len(key); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("%w: must be 16, 24, or 32 bytes, got %d", ErrInvalidKey, l)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	return &Encryptor{gcm: gcm}, nil
}

// NewEncryptorFromConfig decodes a key from configuration. Both hex and base64
// (standard) encodings are accepted; the decoded key must be a valid AES
// length. An empty key is rejected so tokens are never persisted plaintext.
func NewEncryptorFromConfig(encoded string) (*Encryptor, error) {
	if encoded == "" {
		return nil, fmt.Errorf("%w: encryption key is required", ErrInvalidKey)
	}
	for _, dec := range []func(string) ([]byte, error){
		hex.DecodeString,
		func(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) },
	} {
		if raw, derr := dec(encoded); derr == nil {
			if enc, nerr := NewEncryptor(raw); nerr == nil {
				return enc, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: key must be 16/24/32-byte hex or base64", ErrInvalidKey)
}

// Encrypt seals plaintext with AES-GCM and returns base64(nonce || ciphertext).
// Empty input round-trips to an empty string (no nonce is generated).
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt, failing closed on any authentication or decode
// error so a tampered or unreadable token is never returned as plaintext.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
	}
	ns := e.gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("%w: ciphertext shorter than nonce", ErrInvalidCiphertext)
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := e.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCiphertext, err)
	}
	return string(plain), nil
}
