package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// defaultAuthorizeURL is the public Bitbucket OAuth authorization endpoint.
	defaultAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"
	// defaultTokenURL is the public Bitbucket OAuth token endpoint.
	defaultTokenURL = "https://bitbucket.org/site/oauth2/access_token" //nolint:gosec // endpoint path, not a credential
	// maxTokenResponseBytes caps the token endpoint response body.
	maxTokenResponseBytes = 1 << 20
)

// BitbucketClient exchanges OAuth authorization codes and refreshes tokens
// against the Bitbucket OAuth token endpoint, and builds authorization URLs.
// It never logs or serializes raw tokens.
type BitbucketClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	authorizeURL string
	tokenURL     string
	http         *http.Client
}

// NewBitbucketClient creates the Bitbucket OAuth client. Empty endpoint URLs
// fall back to the public Bitbucket OAuth endpoints.
func NewBitbucketClient(cfg Config) *BitbucketClient {
	cfg = normalizeConfig(cfg)

	return &BitbucketClient{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURI:  cfg.RedirectURI,
		authorizeURL: cfg.AuthorizeURL,
		tokenURL:     cfg.TokenURL,
		http:         http.DefaultClient,
	}
}

// Configured reports whether the OAuth app credentials needed for the
// authorization code flow are present.
func (c *BitbucketClient) Configured() bool {
	return c.clientID != "" && c.redirectURI != ""
}

// RedirectURI returns the configured OAuth callback URL.
func (c *BitbucketClient) RedirectURI() string {
	return c.redirectURI
}

// AuthorizeURL builds the Bitbucket authorization URL for the given CSRF
// state. The requested scope is the mandatory minimum 'webhook'.
func (c *BitbucketClient) AuthorizeURL(state string) string {
	query := url.Values{}
	query.Set("client_id", c.clientID)
	query.Set("response_type", "code")
	query.Set("scope", RequiredScope)
	query.Set("state", state)
	query.Set("redirect_uri", c.redirectURI)

	return c.authorizeURL + "?" + query.Encode()
}

// Exchange exchanges an authorization code for access and refresh tokens.
func (c *BitbucketClient) Exchange(ctx context.Context, code string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURI)

	return c.requestToken(ctx, form)
}

// Refresh exchanges a refresh token for a fresh access token. Bitbucket Cloud
// refresh tokens are single-use; every response carries a new one.
func (c *BitbucketClient) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	return c.requestToken(ctx, form)
}

// tokenResponse mirrors the Bitbucket OAuth token endpoint payload.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
}

// requestToken performs an OAuth token grant. The client authenticates with
// HTTP Basic credentials. Errors never include upstream bodies or tokens.
func (c *BitbucketClient) requestToken(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read oauth token response: %w", err)
	}

	var parsed tokenResponse
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse oauth token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || parsed.AccessToken == "" {
		return nil, fmt.Errorf("%w: Bitbucket returned status %d", ErrTokenExchangeFailed, resp.StatusCode)
	}

	token := &Token{
		AccessToken:       parsed.AccessToken,
		RefreshToken:      parsed.RefreshToken,
		Scope:             parsed.Scopes,
		ExpiresAt:         time.Time{},
		ConnectedByUserID: 0,
		CreatedAt:         time.Time{},
		UpdatedAt:         time.Time{},
	}
	if parsed.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	}

	return token, nil
}
