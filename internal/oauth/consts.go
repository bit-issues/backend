package oauth

import "time"

const (
	// defaultAuthorizeURL is the public Bitbucket OAuth authorization endpoint.
	defaultAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"
	// defaultTokenURL is the public Bitbucket OAuth token endpoint.
	defaultTokenURL = "https://bitbucket.org/site/oauth2/access_token" //nolint:gosec // endpoint path, not a credential
	// requiredScope is the minimum Bitbucket OAuth scope for webhook management.
	requiredScope = "webhook"

	// defaultRefreshThreshold is the maximum time between token refreshes.
	defaultRefreshThreshold = 15 * time.Minute
)
