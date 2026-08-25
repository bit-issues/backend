package oauth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	serveroauth "github.com/bit-issues/backend/internal/server/oauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/fiberfx"
	"github.com/gofiber/fiber/v2"
	jwtx "github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"go.uber.org/zap"
)

const (
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
	testRedirectURI  = "https://issues.example.com/api/v1/oauth/bitbucket/callback"
	testJWTSecret    = "test-jwt-secret"
	testJWTIssuer    = "test"

	testAccessTokenValue  = "super-secret-access-token"
	testRefreshTokenValue = "super-secret-refresh-token"

	adminUserID   int64 = 1
	regularUserID int64 = 2

	refreshTTL = 24 * time.Hour
)

// newOAuthService returns an oauth service wired to a sqlmock-backed
// repository, mirroring the production singleflight-free read paths used by
// the endpoints under test.
func newOAuthService(t *testing.T) (*oauth.Service, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	sqldb.SetMaxOpenConns(1)

	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version()"}).AddRow("8.0.36"))

	db := bun.NewDB(sqldb, mysqldialect.New())

	t.Cleanup(func() {
		_ = sqldb.Close()
	})

	return oauth.NewService(oauth.Config{}, oauth.NewRepository(db), nil, zap.NewNop()), mock
}

// newUsersDB returns a bun.DB for the users repository (sqlmock-backed). The
// jwtauth middleware looks the authenticated user up on every admin route.
func newUsersDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	sqldb.SetMaxOpenConns(1)

	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version()"}).AddRow("8.0.36"))

	db := bun.NewDB(sqldb, mysqldialect.New())

	t.Cleanup(func() {
		_ = sqldb.Close()
	})

	return db, mock
}

// newTestApp builds the production-like Fiber app: the public callback is
// registered before the JWT middleware, the admin routes after it.
func newTestApp(t *testing.T, svc *oauth.Service, client *oauth.BitbucketClient, usersDB *bun.DB) *fiber.App {
	t.Helper()

	usersSvc := users.NewService(users.NewRepository(usersDB))
	jwtSvc := jwt.NewService(jwt.Config{
		Secret:     testJWTSecret,
		AccessTTL:  time.Minute,
		RefreshTTL: refreshTTL,
		Issuer:     testJWTIssuer,
	}, nil)

	app := fiber.New(fiber.Config{ErrorHandler: fiberfx.NewJSONErrorHandler(zap.NewNop())})
	v1 := app.Group("/api/v1")

	handler := serveroauth.NewHandler(svc, client, zap.NewNop())
	handler.RegisterPublic(v1)

	v1.Use(jwtauth.New(jwtSvc, usersSvc), jwtauth.ErrorsHandler())

	handler.Register(v1)

	return app
}

// newTestClient builds the Bitbucket OAuth client against the given token
// endpoint override.
func newTestClient(tokenURL string) *oauth.BitbucketClient {
	return oauth.NewBitbucketClient(oauth.Config{
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		RedirectURI:  testRedirectURI,
		TokenURL:     tokenURL,
	})
}

// tokenRequest captures the token endpoint call for assertions.
type tokenRequest struct {
	method      string
	path        string
	contentType string
	auth        string
	form        url.Values
	count       int
}

// newTokenServer runs a mock Bitbucket OAuth token endpoint.
func newTokenServer(t *testing.T, status int, body string, capture *tokenRequest) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			capture.count++
			capture.method = r.Method
			capture.path = r.URL.Path
			capture.contentType = r.Header.Get("Content-Type")
			capture.auth = r.Header.Get("Authorization")
			_ = r.ParseForm()
			capture.form = r.PostForm
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// tokenBody returns a Bitbucket-style successful token response.
func tokenBody(scope string) string {
	return `{"access_token":"` + testAccessTokenValue +
		`","refresh_token":"` + testRefreshTokenValue +
		`","scopes":"` + scope + `","expires_in":7200}`
}

// createState stores a CSRF state for the admin user and returns it.
func createState(t *testing.T, svc *oauth.Service, mock sqlmock.Sqlmock) string {
	t.Helper()

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	state, err := svc.CreateState(context.Background(), adminUserID, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}

	return state
}

// expectStateConsumption registers the lookup and single-use deletion of a
// valid CSRF state.
func expectStateConsumption(mock sqlmock.Sqlmock, state string) {
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), adminUserID, testRedirectURI,
			time.Now().Add(10*time.Minute), time.Now(),
		))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// expectTokenSave registers the singleton upsert of the OAuth credential.
func expectTokenSave(mock sqlmock.Sqlmock) {
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectUserLookup registers the jwtauth user resolution.
func expectUserLookup(mock sqlmock.Sqlmock, id int64, role users.Role) {
	mock.ExpectQuery("(?i)SELECT .* FROM `users`").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "name", "password_hash", "role", "status", "created_at", "updated_at",
		}).AddRow(
			id, "admin@example.com", "Admin", "hash", string(role), string(users.StatusActive),
			time.Now(), time.Now(),
		))
}

// checkMocks asserts every registered sqlmock expectation was consumed.
func checkMocks(t *testing.T, mocks ...sqlmock.Sqlmock) {
	t.Helper()

	for _, mock := range mocks {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	}
}

func hashState(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

// makeToken signs an access token for the given user, mirroring the claims
// the JWT service expects.
func makeToken(t *testing.T, userID int64, role users.Role) string {
	t.Helper()

	now := time.Now()
	claims := jwt.Claims{
		RegisteredClaims: jwtx.RegisteredClaims{
			Issuer:    testJWTIssuer,
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwtx.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwtx.NewNumericDate(now),
			IssuedAt:  jwtx.NewNumericDate(now),
		},
		UserID: userID,
		Role:   role,
		Status: users.StatusActive,
	}

	token := jwtx.NewWithClaims(jwtx.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return signed
}

func adminToken(t *testing.T) string {
	t.Helper()

	return makeToken(t, adminUserID, users.RoleAdmin)
}

func doRequest(t *testing.T, app *fiber.App, method, path string, token string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
}

func TestAuthorizeReturnsAuthorizationURL(t *testing.T) {
	svc, oauthMock := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, adminUserID, users.RoleAdmin)
	oauthMock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/authorize", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, fiber.StatusOK, body)
	}

	var out serveroauth.AuthorizeResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.URL == "" {
		t.Fatal("url is empty")
	}

	u, err := url.Parse(out.URL)
	if err != nil {
		t.Fatalf("authorize URL parse: %v", err)
	}
	if u.Scheme != "https" || u.Host != "bitbucket.org" || u.Path != "/site/oauth2/authorize" {
		t.Errorf("authorize URL = %q, want https://bitbucket.org/site/oauth2/authorize", out.URL)
	}

	q := u.Query()
	for key, want := range map[string]string{
		"client_id":     testClientID,
		"response_type": "code",
		"scope":         oauth.RequiredScope,
		"redirect_uri":  testRedirectURI,
	} {
		if got := q.Get(key); got != want {
			t.Errorf("authorize query[%q] = %q, want %q", key, got, want)
		}
	}

	raw, err := base64.RawURLEncoding.DecodeString(q.Get("state"))
	if err != nil {
		t.Fatalf("state is not base64url: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("state length = %d bytes, want 32", len(raw))
	}

	checkMocks(t, usersMock, oauthMock)
}

func TestAuthorizeFailsWhenOAuthNotConfigured(t *testing.T) {
	svc, oauthMock := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, oauth.NewBitbucketClient(oauth.Config{}), usersDB)

	expectUserLookup(usersMock, adminUserID, users.RoleAdmin)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/authorize", adminToken(t))
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}

	// No CSRF state may be created when the OAuth app is not configured.
	checkMocks(t, usersMock, oauthMock)
}

func TestAuthorizeRequiresAdminRole(t *testing.T) {
	svc, _ := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, regularUserID, users.RoleUser)

	resp := doRequest(
		t,
		app,
		http.MethodGet,
		"/api/v1/oauth/bitbucket/authorize",
		makeToken(t, regularUserID, users.RoleUser),
	)
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusForbidden)
	}
	checkMocks(t, usersMock)
}

func TestAuthorizeRequiresAuthentication(t *testing.T) {
	svc, _ := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/authorize", "")
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestCallbackExchangesCodeAndConnects(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook repository:admin"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL+"/site/oauth2/access_token"), usersDB)

	state := createState(t, svc, oauthMock)
	expectStateConsumption(oauthMock, state)
	expectTokenSave(oauthMock)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=auth-code-123&state="+state, "")
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=success" {
		t.Errorf("Location = %q, want %q", loc, "/admin?oauth=success")
	}

	if captured.count != 1 {
		t.Errorf("token endpoint calls = %d, want 1", captured.count)
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
		"code":         "auth-code-123",
		"redirect_uri": testRedirectURI,
	} {
		if got := captured.form.Get(key); got != want {
			t.Errorf("form[%q] = %q, want %q", key, got, want)
		}
	}

	for _, leak := range []string{testAccessTokenValue, testRefreshTokenValue} {
		if strings.Contains(body, leak) || strings.Contains(resp.Header.Get("Location"), leak) {
			t.Error("response leaked a token value")
		}
	}

	checkMocks(t, oauthMock)
}

func TestCallbackRejectsReplayedState(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	state := createState(t, svc, oauthMock)
	expectStateConsumption(oauthMock, state)
	expectTokenSave(oauthMock)

	first := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=code-1&state="+state, "")
	_ = readBody(t, first)
	if loc := first.Header.Get("Location"); loc != "/admin?oauth=success" {
		t.Fatalf("first callback Location = %q, want success", loc)
	}

	// The state was deleted on first use; the replay must fail before any
	// second token exchange happens.
	oauthMock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	second := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=code-2&state="+state, "")
	_ = readBody(t, second)

	if second.StatusCode != fiber.StatusFound {
		t.Errorf("replay status = %d, want %d", second.StatusCode, fiber.StatusFound)
	}
	if loc := second.Header.Get("Location"); loc != "/admin?oauth=error&reason=invalid_state" {
		t.Errorf("replay Location = %q, want invalid_state", loc)
	}
	if captured.count != 1 {
		t.Errorf("token endpoint calls = %d, want 1 (no exchange on replay)", captured.count)
	}

	checkMocks(t, oauthMock)
}

func TestCallbackRejectsUnknownState(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	oauthMock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=code-1&state=forged-state", "")
	_ = readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=invalid_state" {
		t.Errorf("Location = %q, want invalid_state", loc)
	}
	if captured.count != 0 {
		t.Errorf("token endpoint calls = %d, want 0", captured.count)
	}

	checkMocks(t, oauthMock)
}

func TestCallbackRejectsMissingParams(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook"), &captured)
	svc, _ := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback", "")
	_ = readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=missing_params" {
		t.Errorf("Location = %q, want missing_params", loc)
	}
	if captured.count != 0 {
		t.Errorf("token endpoint calls = %d, want 0", captured.count)
	}
}

func TestCallbackConsumesStateWhenCodeMissing(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	state := createState(t, svc, oauthMock)
	// A supplied state is still consumed (single-use) even though the code is
	// missing, so it can never be replayed against a later callback.
	expectStateConsumption(oauthMock, state)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?state="+state, "")
	_ = readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=missing_params" {
		t.Errorf("Location = %q, want missing_params", loc)
	}
	if captured.count != 0 {
		t.Errorf("token endpoint calls = %d, want 0", captured.count)
	}

	checkMocks(t, oauthMock)
}

func TestCallbackHandlesBitbucketAccessDenied(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("webhook"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	state := createState(t, svc, oauthMock)
	// The state is still consumed on denial so it can never be replayed.
	expectStateConsumption(oauthMock, state)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?error=access_denied&state="+state, "")
	_ = readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=access_denied" {
		t.Errorf("Location = %q, want access_denied", loc)
	}
	if captured.count != 0 {
		t.Errorf("token endpoint calls = %d, want 0", captured.count)
	}

	checkMocks(t, oauthMock)
}

func TestCallbackHandlesExchangeFailure(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(
		t,
		http.StatusBadRequest,
		`{"error":"invalid_grant","error_description":"bad code"}`,
		&captured,
	)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	state := createState(t, svc, oauthMock)
	expectStateConsumption(oauthMock, state)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=bad-code&state="+state, "")
	body := readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=exchange_failed" {
		t.Errorf("Location = %q, want exchange_failed", loc)
	}
	for _, leak := range []string{"invalid_grant", "bad code", testAccessTokenValue, testRefreshTokenValue} {
		if strings.Contains(body, leak) || strings.Contains(resp.Header.Get("Location"), leak) {
			t.Error("response leaked a sensitive value")
		}
	}

	checkMocks(t, oauthMock)
}

func TestCallbackRejectsMissingWebhookScope(t *testing.T) {
	var captured tokenRequest
	server := newTokenServer(t, http.StatusOK, tokenBody("repository:admin"), &captured)
	svc, oauthMock := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(server.URL), usersDB)

	state := createState(t, svc, oauthMock)
	expectStateConsumption(oauthMock, state)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/callback?code=code-1&state="+state, "")
	_ = readBody(t, resp)

	if loc := resp.Header.Get("Location"); loc != "/admin?oauth=error&reason=invalid_scope" {
		t.Errorf("Location = %q, want invalid_scope", loc)
	}
	if captured.count != 1 {
		t.Errorf("token endpoint calls = %d, want 1", captured.count)
	}

	checkMocks(t, oauthMock)
}

func TestStatusWhenNotConnected(t *testing.T) {
	svc, oauthMock := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, adminUserID, users.RoleAdmin)
	oauthMock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(sqlmock.NewRows([]string{"singleton_id"}))

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/status", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var out serveroauth.StatusResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Connected {
		t.Error("Connected = true, want false")
	}
	if out.ConnectedAt != nil || out.ExpiresAt != nil || out.Scopes != nil {
		t.Errorf("optional fields must be empty when disconnected: %+v", out)
	}

	checkMocks(t, usersMock, oauthMock)
}

func TestStatusWhenConnected(t *testing.T) {
	now := time.Now()
	svc, oauthMock := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, adminUserID, users.RoleAdmin)
	oauthMock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(sqlmock.NewRows([]string{
			"singleton_id", "access_token", "refresh_token", "scope",
			"expires_at", "connected_by_user_id", "created_at", "updated_at",
		}).AddRow(
			oauth.SingletonID, "access-token", "refresh-token", "webhook repository:admin",
			now.Add(time.Hour), adminUserID, now.Add(-time.Hour), now.Add(-time.Hour),
		))

	resp := doRequest(t, app, http.MethodGet, "/api/v1/oauth/bitbucket/status", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	var out serveroauth.StatusResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !out.Connected {
		t.Error("Connected = false, want true")
	}
	if out.ConnectedAt == nil || *out.ConnectedAt == "" {
		t.Error("ConnectedAt is missing")
	}
	if out.ExpiresAt == nil || *out.ExpiresAt == "" {
		t.Error("ExpiresAt is missing")
	}
	wantScopes := []string{"webhook", "repository:admin"}
	if len(out.Scopes) != len(wantScopes) {
		t.Fatalf("Scopes = %v, want %v", out.Scopes, wantScopes)
	}
	for i := range wantScopes {
		if out.Scopes[i] != wantScopes[i] {
			t.Errorf("Scopes = %v, want %v", out.Scopes, wantScopes)
		}
	}
	if strings.Contains(body, "access-token") || strings.Contains(body, "refresh-token") {
		t.Error("status response leaks token values")
	}

	checkMocks(t, usersMock, oauthMock)
}

func TestStatusRequiresAdminRole(t *testing.T) {
	svc, _ := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, regularUserID, users.RoleUser)

	resp := doRequest(
		t,
		app,
		http.MethodGet,
		"/api/v1/oauth/bitbucket/status",
		makeToken(t, regularUserID, users.RoleUser),
	)
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusForbidden)
	}
	checkMocks(t, usersMock)
}

func TestDisconnectRemovesTokens(t *testing.T) {
	svc, oauthMock := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, adminUserID, users.RoleAdmin)
	oauthMock.ExpectExec("(?i)DELETE FROM `oauth_tokens`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp := doRequest(t, app, http.MethodPost, "/api/v1/oauth/bitbucket/disconnect", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", resp.StatusCode, fiber.StatusOK, body)
	}

	checkMocks(t, usersMock, oauthMock)
}

func TestDisconnectRequiresAdminRole(t *testing.T) {
	svc, _ := newOAuthService(t)
	usersDB, usersMock := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	expectUserLookup(usersMock, regularUserID, users.RoleUser)

	resp := doRequest(
		t,
		app,
		http.MethodPost,
		"/api/v1/oauth/bitbucket/disconnect",
		makeToken(t, regularUserID, users.RoleUser),
	)
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusForbidden)
	}
	checkMocks(t, usersMock)
}

func TestOAuthRoutesRequireAuthentication(t *testing.T) {
	svc, _ := newOAuthService(t)
	usersDB, _ := newUsersDB(t)
	app := newTestApp(t, svc, newTestClient(""), usersDB)

	for _, route := range []struct{ method, path string }{
		{method: http.MethodGet, path: "/api/v1/oauth/bitbucket/authorize"},
		{method: http.MethodGet, path: "/api/v1/oauth/bitbucket/status"},
		{method: http.MethodPost, path: "/api/v1/oauth/bitbucket/disconnect"},
	} {
		resp := doRequest(t, app, route.method, route.path, "")
		_ = readBody(t, resp)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", route.method, route.path, resp.StatusCode, fiber.StatusUnauthorized)
		}
	}
}
