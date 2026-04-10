package jwt

import (
	"fmt"
	"time"
)

const (
	minSecretLength = 32
)

type Config struct {
	Secret    string
	AccessTTL time.Duration
	Issuer    string
}

func (c Config) Validate() error {
	if c.Secret == "" {
		return fmt.Errorf("%w: secret is required", ErrInvalidConfig)
	}

	if len(c.Secret) < minSecretLength {
		return fmt.Errorf("%w: secret must be at least %d bytes", ErrInvalidConfig, minSecretLength)
	}

	if c.AccessTTL <= 0 {
		return fmt.Errorf("%w: access ttl must be positive", ErrInvalidConfig)
	}

	return nil
}
