package oauth

// Config holds OAuth service tunables. Zero values fall back to the defaults
// from DefaultConfig.
type Config struct {
	// ClientID is the Bitbucket OAuth app consumer key.
	ClientID string
	// ClientSecret is the Bitbucket OAuth app consumer secret. Never logged.
	ClientSecret string
}
