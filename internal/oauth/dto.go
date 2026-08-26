package oauth

// tokenResponse mirrors the Bitbucket OAuth token endpoint payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
}
