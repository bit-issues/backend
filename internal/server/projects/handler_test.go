package projects_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	serverprojects "github.com/bit-issues/backend/internal/server/projects"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"github.com/go-core-fx/fiberfx"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	jwtx "github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"go.uber.org/zap"
)

const (
	testAccessToken   = "test-access-token"
	testWebhookSecret = "test-webhook-secret"
	testCallbackURL   = "https://issues.example.com"
	testCallbackPath  = "/api/v1/webhooks/bitbucket/push"
	testJWTSecret     = "test-jwt-secret"
	testJWTIssuer     = "test"
	testProjectSlug   = "my-project"
	testRepoURL       = "https://bitbucket.org/workspace/repo-slug"
	testHookUUID      = "{abc-123}"
	testHookCreatedAt = "2026-08-20T10:00:00+00:00"
	testHookPath      = "/2.0/repositories/workspace/repo-slug/hooks"

	adminUserID   int64 = 1
	regularUserID int64 = 2

	refreshTTL = 24 * time.Hour
)

// stubRows is a single-row [driver.Rows] implementation.
type stubRows struct {
	cols   []string
	values []driver.Value
	done   bool
}

func (r *stubRows) Columns() []string { return r.cols }
func (r *stubRows) Close() error      { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	copy(dest, r.values)

	return nil
}

// stubConn serves the few queries the test doubles issue: the MySQL version
// probe, the project lookup, and the user lookup. Every query is recorded so
// tests can assert that webhook state never touches the database.
type stubConn struct {
	queries *[]string
}

func (c *stubConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *stubConn) Close() error { return nil }

func (c *stubConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}
func (c *stubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	*c.queries = append(*c.queries, query)

	switch {
	case strings.Contains(query, "version()"):
		return &stubRows{cols: []string{"version()"}, values: []driver.Value{"8.0.36"}}, nil
	case strings.Contains(query, "FROM `projects`"):
		return projectRow(), nil
	case strings.Contains(query, "FROM `users`"):
		return userRow(query), nil
	}

	return nil, fmt.Errorf("unexpected query: %s", query)
}

func projectRow() driver.Rows {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	return &stubRows{
		cols:   []string{"id", "name", "repo_url", "created_at", "updated_at"},
		values: []driver.Value{testProjectSlug, "My Project", testRepoURL, now, now},
	}
}

func userRow(query string) driver.Rows {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	role := "admin"
	if strings.Contains(query, "id = 2") {
		role = "user"
	}

	return &stubRows{
		cols:   []string{"id", "email", "name", "password_hash", "role", "status", "created_at", "updated_at"},
		values: []driver.Value{adminUserID, "admin@example.com", "Admin", "hash", role, "active", now, now},
	}
}

type stubConnector struct {
	conn *stubConn
}

func (c stubConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c stubConnector) Driver() driver.Driver                        { return nil }

// newStubBunDB returns a bun.DB backed by the recording stub driver.
func newStubBunDB(queries *[]string) *bun.DB {
	db := sql.OpenDB(stubConnector{conn: &stubConn{queries: queries}})

	return bun.NewDB(db, mysqldialect.New())
}

// newTestApp builds the production-like Fiber app: jwtauth middleware, the
// projects handler, and the JSON error handler.
func newTestApp(t *testing.T, bbClient *bitbucket.Client, secret string, queries *[]string) *fiber.App {
	t.Helper()

	return newTestAppWithOAuth(t, bbClient, secret, queries, nil)
}

// newTestAppWithOAuth builds the production-like Fiber app with OAuth token
// resolution wired into the webhook management service.
func newTestAppWithOAuth(
	t *testing.T,
	bbClient *bitbucket.Client,
	secret string,
	queries *[]string,
	oauthSvc *oauth.Service,
) *fiber.App {
	t.Helper()

	db := newStubBunDB(queries)
	projectsSvc := projects.NewService(projects.NewRepository(db))
	webhooksSvc := webhooks.NewManagementService(
		webhooks.Config{Secret: secret},
		bitbucket.Config{AccessToken: testAccessToken, CallbackURL: testCallbackURL},
		bbClient,
	)
	if oauthSvc != nil {
		webhooksSvc.SetOAuthService(oauthSvc)
	}
	usersSvc := users.NewService(users.NewRepository(db))
	jwtSvc := jwt.NewService(jwt.Config{
		Secret:     testJWTSecret,
		AccessTTL:  time.Minute,
		RefreshTTL: refreshTTL,
		Issuer:     testJWTIssuer,
	}, nil)

	app := fiber.New(fiber.Config{ErrorHandler: fiberfx.NewJSONErrorHandler(zap.NewNop())})
	v1 := app.Group("/api/v1", jwtauth.New(jwtSvc, usersSvc), jwtauth.ErrorsHandler())
	serverprojects.NewHandler(projectsSvc, webhooksSvc, nil, nil, validator.New()).Register(v1)

	return app
}

// newOAuthService returns an oauth service backed by a sqlmock repository.
func newOAuthService(t *testing.T, refresher oauth.TokenRefresher) (*oauth.Service, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	sqldb.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqldb.Close()
	})

	// bun's mysqldialect probes the server version inside bun.NewDB, before
	// test expectations are registered. Consume it up front.
	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version()"}).AddRow("8.0.36"))

	db := bun.NewDB(sqldb, mysqldialect.New())

	return oauth.NewService(oauth.Config{}, oauth.NewRepository(db), refresher, zap.NewNop()), mock
}

// oauthTokenRow mirrors the oauth_tokens row shape used by the repository.
func oauthTokenRow(token *oauth.Token) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"singleton_id", "access_token", "refresh_token", "scope",
		"expires_at", "connected_by_user_id", "created_at", "updated_at",
	}).AddRow(
		oauth.SingletonID, token.AccessToken, token.RefreshToken, token.Scope,
		token.ExpiresAt, token.ConnectedByUserID, token.CreatedAt, token.UpdatedAt,
	)
}

// revokedOAuthService returns a service whose stored OAuth credential cannot
// be refreshed, simulating a revoked or invalid token.
func revokedOAuthService(t *testing.T) *oauth.Service {
	t.Helper()

	now := time.Now()
	svc, mock := newOAuthService(t, func(context.Context, string) (*oauth.Token, error) {
		return nil, oauth.ErrTokenExchangeFailed
	})
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(oauthTokenRow(&oauth.Token{
			AccessToken:       "expired-access-token",
			RefreshToken:      "expired-refresh-token",
			Scope:             "webhook",
			ExpiresAt:         now.Add(-30 * time.Minute),
			ConnectedByUserID: adminUserID,
			CreatedAt:         now.Add(-3 * time.Hour),
			UpdatedAt:         now.Add(-3 * time.Hour),
		}))
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	return svc
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

func decodeStatus(t *testing.T, resp *http.Response) serverprojects.WebhookStatusResponse {
	t.Helper()

	var status serverprojects.WebhookStatusResponse
	if err := json.Unmarshal([]byte(readBody(t, resp)), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}

	return status
}

// newBitbucketClient builds a real client against a mock Bitbucket server.
func newBitbucketClient(t *testing.T, handler http.Handler) *bitbucket.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     server.URL,
		AccessToken: testAccessToken,
		CallbackURL: testCallbackURL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	return client
}

func hookJSON(description string) map[string]any {
	return map[string]any{
		"uuid":        testHookUUID,
		"url":         testCallbackURL + testCallbackPath,
		"description": description,
		"active":      true,
		"events":      []string{"repo:push"},
		"created_at":  testHookCreatedAt,
		"secret_set":  true,
	}
}

func adminToken(t *testing.T) string {
	t.Helper()

	return makeToken(t, adminUserID, users.RoleAdmin)
}

// assertNoWebhookDBUsage fails if any recorded query writes to the database
// or references webhook state; the stateless architecture forbids both.
func assertNoWebhookDBUsage(t *testing.T, queries []string) {
	t.Helper()

	for _, q := range queries {
		lower := strings.ToLower(q)
		if strings.HasPrefix(lower, "insert") ||
			strings.HasPrefix(lower, "update") ||
			strings.HasPrefix(lower, "delete") ||
			strings.Contains(lower, "webhook") {
			t.Errorf("webhook operation touched the database: %s", q)
		}
	}
}

func TestRegisterWebhookCreatesWhenAbsent(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuth    string
		gotPayload map[string]any
	)

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case http.MethodPost:
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			if unmarshalErr := json.Unmarshal(body, &gotPayload); unmarshalErr != nil {
				t.Errorf("unmarshal body: %v", unmarshalErr)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(hookJSON("BitIssues webhook"))
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/projects/"+testProjectSlug+"/webhook/register", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusRegistered)
	}
	if status.HookUUID != testHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, testHookUUID)
	}
	if status.CallbackURL != testCallbackURL+testCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, testCallbackURL+testCallbackPath)
	}
	if status.UpdatedAt != testHookCreatedAt {
		t.Errorf("UpdatedAt = %q, want %q", status.UpdatedAt, testHookCreatedAt)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != testHookPath {
		t.Errorf("path = %q, want %q", gotPath, testHookPath)
	}
	if gotAuth != "Bearer "+testAccessToken {
		t.Errorf("Authorization = %q, want Bearer %q", gotAuth, testAccessToken)
	}

	for field, want := range map[string]any{
		"description": "BitIssues webhook",
		"url":         testCallbackURL + testCallbackPath,
		"active":      true,
		"secret":      testWebhookSecret,
	} {
		if got := gotPayload[field]; got != want {
			t.Errorf("payload[%q] = %v, want %v", field, got, want)
		}
	}
	events, ok := gotPayload["events"].([]any)
	if !ok || len(events) != 1 || events[0] != "repo:push" {
		t.Errorf("payload[events] = %#v, want [repo:push]", gotPayload["events"])
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestRegisterWebhookUpdatesExistingHook(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotPayload map[string]any
	)

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{hookJSON("existing hook")}})
		case http.MethodPut:
			gotMethod = r.Method
			gotPath = r.URL.Path

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			if unmarshalErr := json.Unmarshal(body, &gotPayload); unmarshalErr != nil {
				t.Errorf("unmarshal body: %v", unmarshalErr)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hookJSON("existing hook"))
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/projects/"+testProjectSlug+"/webhook/register", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusRegistered)
	}
	if status.HookUUID != testHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, testHookUUID)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != testHookPath+"/"+testHookUUID {
		t.Errorf("path = %q, want %q", gotPath, testHookPath+"/"+testHookUUID)
	}

	if gotPayload["description"] != "existing hook" {
		t.Errorf("payload[description] = %v, want existing hook preserved", gotPayload["description"])
	}
	if gotPayload["url"] != testCallbackURL+testCallbackPath {
		t.Errorf("payload[url] = %v, want %q", gotPayload["url"], testCallbackURL+testCallbackPath)
	}
	if gotPayload["active"] != true {
		t.Errorf("payload[active] = %v, want true", gotPayload["active"])
	}
	if gotPayload["secret"] != testWebhookSecret {
		t.Errorf("payload[secret] = %v, want configured secret", gotPayload["secret"])
	}
	events, ok := gotPayload["events"].([]any)
	if !ok || len(events) != 1 || events[0] != "repo:push" {
		t.Errorf("payload[events] = %#v, want [repo:push]", gotPayload["events"])
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestUnregisterWebhookDeletesWhenPresent(t *testing.T) {
	var gotDeletePath string

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{hookJSON("BitIssues webhook")}})
		case http.MethodDelete:
			gotDeletePath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/projects/"+testProjectSlug+"/webhook/unregister", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusNotRegistered)
	}
	if status.HookUUID != "" {
		t.Errorf("HookUUID = %q, want empty", status.HookUUID)
	}
	if status.CallbackURL != testCallbackURL+testCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, testCallbackURL+testCallbackPath)
	}
	if gotDeletePath != testHookPath+"/"+testHookUUID {
		t.Errorf("delete path = %q, want %q", gotDeletePath, testHookPath+"/"+testHookUUID)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestUnregisterWebhookNoOpWhenAbsent(t *testing.T) {
	requests := 0

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/projects/"+testProjectSlug+"/webhook/unregister", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusNotRegistered)
	}
	if requests != 1 {
		t.Errorf("bitbucket requests = %d, want 1 (list only)", requests)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestGetWebhookStatusRegistered(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{hookJSON("BitIssues webhook")}})
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusRegistered)
	}
	if status.HookUUID != testHookUUID {
		t.Errorf("HookUUID = %q, want %q", status.HookUUID, testHookUUID)
	}
	if status.UpdatedAt != testHookCreatedAt {
		t.Errorf("UpdatedAt = %q, want %q", status.UpdatedAt, testHookCreatedAt)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestGetWebhookStatusNotRegistered(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	status := decodeStatus(t, resp)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if status.Status != webhooks.WebhookStatusNotRegistered {
		t.Errorf("Status = %q, want %q", status.Status, webhooks.WebhookStatusNotRegistered)
	}
	if status.HookUUID != "" {
		t.Errorf("HookUUID = %q, want empty", status.HookUUID)
	}
	if status.CallbackURL != testCallbackURL+testCallbackPath {
		t.Errorf("CallbackURL = %q, want %q", status.CallbackURL, testCallbackURL+testCallbackPath)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestRegisterWebhookFailsWhenSecretEmpty(t *testing.T) {
	requests := 0

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))

	var queries []string
	app := newTestApp(t, client, "", &queries)

	resp := doRequest(t, app, http.MethodPost, "/api/v1/projects/"+testProjectSlug+"/webhook/register", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if requests != 0 {
		t.Errorf("bitbucket requests = %d, want 0", requests)
	}
	if strings.Contains(body, testWebhookSecret) {
		t.Errorf("response leaked the webhook secret: %s", body)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestGetWebhookStatusMapsBitbucket403ToForbidden(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"access denied"}}`))
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusForbidden)
	}
	if !strings.Contains(body, "insufficient Bitbucket permissions") {
		t.Errorf("body = %s, want human-readable failure reason", body)
	}
	for _, leak := range []string{testAccessToken, testWebhookSecret, "access denied", "api.bitbucket.org"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaked %q: %s", leak, body)
		}
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestGetWebhookStatusMapsOAuthRevokedToUnauthorized(t *testing.T) {
	requests := 0

	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))

	var queries []string
	app := newTestAppWithOAuth(t, client, testWebhookSecret, &queries, revokedOAuthService(t))

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
	}
	if !strings.Contains(body, "reconnect") {
		t.Errorf("body = %s, want actionable reconnect hint", body)
	}
	for _, leak := range []string{testAccessToken, testWebhookSecret, "expired-access-token"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaked %q: %s", leak, body)
		}
	}
	if requests != 0 {
		t.Errorf("bitbucket requests = %d, want 0", requests)
	}

	assertNoWebhookDBUsage(t, queries)
}

func TestGetWebhookStatusMapsBitbucket400ToBadRequest(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"callback url is not reachable"}}`))
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if !strings.Contains(body, "Bitbucket rejected the webhook configuration") {
		t.Errorf("body = %s, want human-readable failure reason", body)
	}
	for _, leak := range []string{testAccessToken, testWebhookSecret, "callback url is not reachable"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaked %q: %s", leak, body)
		}
	}
}

func TestGetWebhookStatusMapsBitbucketServerErrorToBadGateway(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	body := readBody(t, resp)

	if resp.StatusCode != fiber.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusBadGateway)
	}
	if strings.Contains(body, "boom") {
		t.Errorf("response leaked upstream internals: %s", body)
	}
}

func TestGetWebhookStatusMapsInfrastructureErrorToServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client, err := bitbucket.NewClient(bitbucket.Config{
		BaseURL:     baseURL,
		AccessToken: testAccessToken,
		CallbackURL: testCallbackURL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(t, app, http.MethodGet, "/api/v1/projects/"+testProjectSlug+"/webhook", adminToken(t))
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusServiceUnavailable)
	}
}

func TestWebhookRoutesRequireAuthentication(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	for _, path := range []string{
		"/api/v1/projects/" + testProjectSlug + "/webhook",
		"/api/v1/projects/" + testProjectSlug + "/webhook/register",
		"/api/v1/projects/" + testProjectSlug + "/webhook/unregister",
	} {
		resp := doRequest(t, app, http.MethodGet, path, "")
		_ = readBody(t, resp)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("%s status = %d, want %d", path, resp.StatusCode, fiber.StatusUnauthorized)
		}
	}
}

func TestWebhookRoutesRequireAdminRole(t *testing.T) {
	client := newBitbucketClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var queries []string
	app := newTestApp(t, client, testWebhookSecret, &queries)

	resp := doRequest(
		t,
		app,
		http.MethodGet,
		"/api/v1/projects/"+testProjectSlug+"/webhook",
		makeToken(t, regularUserID, users.RoleUser),
	)
	_ = readBody(t, resp)

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusForbidden)
	}
}
