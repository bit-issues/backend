package oauth

// Config holds OAuth service tunables.
type Config struct {
	// ClientID is the Bitbucket OAuth app consumer key.
	ClientID string
	// ClientSecret is the Bitbucket OAuth app consumer secret. Never logged.
	ClientSecret string
	// TokenEncryptionKey is the AES key (hex or base64, 16/24/32 bytes) used to
	// encrypt the stored OAuth access and refresh tokens at rest. Required.
	TokenEncryptionKey string
}

// enabled reports whether OAuth is fully configured. All of the Bitbucket
// client credentials and the token encryption key must be present; otherwise
// the OAuth connection stays disabled and startup must not fail.
func (c Config) enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}
