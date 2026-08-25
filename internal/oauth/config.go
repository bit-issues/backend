package oauth

import "time"

// Config holds OAuth service tunables. Zero values fall back to the defaults
// from DefaultConfig.
type Config struct {
	// StateTTL is how long a CSRF state stays valid (default 10m).
	StateTTL time.Duration
	// AccessTokenLifetime is applied to tokens saved without an expiry
	// (Bitbucket OAuth access tokens live 7200s, default 2h).
	AccessTokenLifetime time.Duration
	// RefreshThreshold triggers proactive refresh when the access token is
	// within this window of expiry (default 15m).
	RefreshThreshold time.Duration
	// ClientID is the Bitbucket OAuth app consumer key.
	ClientID string
	// ClientSecret is the Bitbucket OAuth app consumer secret. Never logged.
	ClientSecret string
	// RedirectURI is the registered Bitbucket OAuth callback URL.
	RedirectURI string
	// AuthorizeURL overrides the Bitbucket OAuth authorization endpoint.
	AuthorizeURL string
	// TokenURL overrides the Bitbucket OAuth token endpoint.
	TokenURL string
}

// DefaultConfig returns the OAuth service defaults mandated by the plan.
func DefaultConfig() Config {
	//nolint:mnd // plan-mandated defaults
	return Config{
		StateTTL:            10 * time.Minute,
		AccessTokenLifetime: 7200 * time.Second,
		RefreshThreshold:    15 * time.Minute,
		ClientID:            "",
		ClientSecret:        "",
		RedirectURI:         "",
		AuthorizeURL:        "",
		TokenURL:            "",
	}
}

func normalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.StateTTL == 0 {
		cfg.StateTTL = def.StateTTL
	}
	if cfg.AccessTokenLifetime == 0 {
		cfg.AccessTokenLifetime = def.AccessTokenLifetime
	}
	if cfg.RefreshThreshold == 0 {
		cfg.RefreshThreshold = def.RefreshThreshold
	}
	if cfg.AuthorizeURL == "" {
		cfg.AuthorizeURL = defaultAuthorizeURL
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = defaultTokenURL
	}
	return cfg
}
