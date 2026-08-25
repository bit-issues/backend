package oauth_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bit-issues/backend/internal/oauth"
)

const (
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
)

// capturedTokenRequest records the token endpoint call for assertions.
type capturedTokenRequest struct {
	mu          sync.Mutex
	method      string
	path        string
	contentType string
	auth        string
	form        url.Values
	count       int
}

func (c *capturedTokenRequest) record(r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.count++
	c.method = r.Method
	c.path = r.URL.Path
	c.contentType = r.Header.Get("Content-Type")
	c.auth = r.Header.Get("Authorization")
	_ = r.ParseForm()
	c.form = r.PostForm
}

// tokenServer runs a mock Bitbucket OAuth token endpoint.
func tokenServer(t *testing.T, status int, body string, capture *capturedTokenRequest) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			capture.record(r)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// newBitbucketClient builds a client against a token endpoint override.
func newBitbucketClient(tokenURL string) *oauth.BitbucketClient {
	return oauth.NewBitbucketClient(oauth.Config{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURI:  testRedirectURI,
		TokenURL:     tokenURL,
	})
}

func TestExchangePostsAuthorizationCodeGrant(t *testing.T) {
	var captured capturedTokenRequest
	server := tokenServer(
		t,
		http.StatusOK,
		`{"access_token":"access-token-value","refresh_token":"refresh-token-value","scopes":"webhook repository:admin","expires_in":7200}`,
		&captured,
	)

	token, err := newBitbucketClient(
		server.URL+"/site/oauth2/access_token",
	).Exchange(context.Background(), "authorization-code-123")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}

	if token.AccessToken != "access-token-value" {
		t.Error("AccessToken mismatch: token value redacted in failure message")
	}
	if token.RefreshToken != "refresh-token-value" {
		t.Error("RefreshToken mismatch: token value redacted in failure message")
	}
	if token.Scope != "webhook repository:admin" {
		t.Errorf("Scope = %q, want %q", token.Scope, "webhook repository:admin")
	}
	wantMin := time.Now().Add(7200*time.Second - 30*time.Second)
	wantMax := time.Now().Add(7200*time.Second + 30*time.Second)
	if token.ExpiresAt.Before(wantMin) || token.ExpiresAt.After(wantMax) {
		t.Errorf("ExpiresAt = %v, want within 30s of now+7200s", token.ExpiresAt)
	}

	if captured.count != 1 {
		t.Fatalf("token endpoint calls = %d, want 1", captured.count)
	}
	if captured.method != http.MethodPost {
		t.Errorf("method = %q, want POST", captured.method)
	}
	if captured.path != "/site/oauth2/access_token" {
		t.Errorf("path = %q, want /site/oauth2/access_token", captured.path)
	}
	if captured.contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", captured.contentType)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(testClientID+":"+testClientSecret))
	if captured.auth != wantAuth {
		t.Errorf("Authorization = %q, want %q", captured.auth, wantAuth)
	}
	for key, want := range map[string]string{
		"grant_type":   "authorization_code",
		"code":         "authorization-code-123",
		"redirect_uri": testRedirectURI,
	} {
		if got := captured.form.Get(key); got != want {
			t.Errorf("form[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestExchangeLeavesZeroExpiryWhenExpiresInAbsent(t *testing.T) {
	var captured capturedTokenRequest
	server := tokenServer(
		t,
		http.StatusOK,
		`{"access_token":"access-token-value","refresh_token":"refresh-token-value","scopes":"webhook"}`,
		&captured,
	)

	token, err := newBitbucketClient(server.URL).Exchange(context.Background(), "code")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if !token.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt = %v, want zero", token.ExpiresAt)
	}
}

func TestExchangeRejectsNon200(t *testing.T) {
	var captured capturedTokenRequest
	server := tokenServer(
		t,
		http.StatusBadRequest,
		`{"error":"invalid_grant","error_description":"the grant is secretly invalid"}`,
		&captured,
	)

	_, err := newBitbucketClient(server.URL).Exchange(context.Background(), "bad-code")
	if !errors.Is(err, oauth.ErrTokenExchangeFailed) {
		t.Fatalf("Exchange() error = %v, want %v", err, oauth.ErrTokenExchangeFailed)
	}
	if strings.Contains(err.Error(), "the grant is secretly invalid") {
		t.Error("error message leaks the upstream error description")
	}
	if strings.Contains(err.Error(), "invalid_grant") {
		t.Error("error message leaks the upstream error code")
	}
	if captured.count != 1 {
		t.Errorf("token endpoint calls = %d, want 1", captured.count)
	}
}

func TestExchangeRejectsMissingAccessToken(t *testing.T) {
	server := tokenServer(t, http.StatusOK, `{"refresh_token":"refresh-token-value","scopes":"webhook"}`, nil)

	_, err := newBitbucketClient(server.URL).Exchange(context.Background(), "code")
	if !errors.Is(err, oauth.ErrTokenExchangeFailed) {
		t.Fatalf("Exchange() error = %v, want %v", err, oauth.ErrTokenExchangeFailed)
	}
}

func TestRefreshPostsRefreshTokenGrant(t *testing.T) {
	var captured capturedTokenRequest
	server := tokenServer(
		t,
		http.StatusOK,
		`{"access_token":"fresh-access-token","refresh_token":"fresh-refresh-token","scopes":"webhook","expires_in":7200}`,
		&captured,
	)

	token, err := newBitbucketClient(server.URL).Refresh(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if token.AccessToken != "fresh-access-token" {
		t.Error("AccessToken mismatch: token value redacted in failure message")
	}
	if token.RefreshToken != "fresh-refresh-token" {
		t.Error("RefreshToken mismatch: token value redacted in failure message")
	}

	if captured.method != http.MethodPost {
		t.Errorf("method = %q, want POST", captured.method)
	}
	if got := captured.form.Get("grant_type"); got != "refresh_token" {
		t.Errorf("form[grant_type] = %q, want refresh_token", got)
	}
	if got := captured.form.Get("refresh_token"); got != "old-refresh-token" {
		t.Error("form[refresh_token] mismatch: token value redacted in failure message")
	}
	if _, ok := captured.form["redirect_uri"]; ok {
		t.Error("refresh grant must not include redirect_uri")
	}
}

func TestRefreshRejectsNon200(t *testing.T) {
	var captured capturedTokenRequest
	server := tokenServer(t, http.StatusForbidden, `{"error":"refresh_token_revoked"}`, &captured)

	_, err := newBitbucketClient(server.URL).Refresh(context.Background(), "revoked-refresh-token")
	if !errors.Is(err, oauth.ErrTokenExchangeFailed) {
		t.Fatalf("Refresh() error = %v, want %v", err, oauth.ErrTokenExchangeFailed)
	}
	if strings.Contains(err.Error(), "refresh_token_revoked") {
		t.Error("error message leaks the upstream error code")
	}
}

func TestAuthorizeURL(t *testing.T) {
	client := newBitbucketClient("")

	raw := client.AuthorizeURL("csrf-state-value")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("AuthorizeURL() parse: %v", err)
	}
	if u.Scheme != "https" || u.Host != "bitbucket.org" || u.Path != "/site/oauth2/authorize" {
		t.Errorf("authorize URL = %q, want https://bitbucket.org/site/oauth2/authorize", raw)
	}

	q := u.Query()
	for key, want := range map[string]string{
		"client_id":     testClientID,
		"response_type": "code",
		"scope":         oauth.RequiredScope,
		"state":         "csrf-state-value",
		"redirect_uri":  testRedirectURI,
	} {
		if got := q.Get(key); got != want {
			t.Errorf("query[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestDefaultsAndConfiguration(t *testing.T) {
	empty := oauth.NewBitbucketClient(oauth.Config{})
	if empty.Configured() {
		t.Error("empty config must not be configured")
	}

	raw := empty.AuthorizeURL("state")
	if !strings.HasPrefix(raw, "https://bitbucket.org/site/oauth2/authorize?") {
		t.Errorf("default authorize URL = %q, want bitbucket.org/site/oauth2/authorize", raw)
	}

	configured := newBitbucketClient("")
	if !configured.Configured() {
		t.Error("full config must be configured")
	}
	if got := configured.RedirectURI(); got != testRedirectURI {
		t.Errorf("RedirectURI() = %q, want %q", got, testRedirectURI)
	}
}

func TestExchangeNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	_, err := newBitbucketClient(baseURL).Exchange(context.Background(), "code")
	if err == nil {
		t.Fatal("Exchange() error = nil, want network error")
	}
	if errors.Is(err, oauth.ErrTokenExchangeFailed) {
		t.Error("network errors must not be reported as exchange failures")
	}
}
