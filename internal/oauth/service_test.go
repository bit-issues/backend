package oauth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bit-issues/backend/internal/oauth"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"go.uber.org/zap"
)

const (
	testRedirectURI = "https://bitissues.example.com/oauth/callback"
	testStateTTL    = 10 * time.Minute
)

// newTestRepo returns an oauth repository backed by sqlmock. The connection
// pool is capped at one so expectation order stays deterministic.
func newTestRepo(t *testing.T) (*oauth.Repository, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	sqldb.SetMaxOpenConns(1)

	// bun's mysqldialect probes the server version inside bun.NewDB, before
	// test expectations are registered. Consume it up front.
	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version()"}).AddRow("8.0.36"))

	db := bun.NewDB(sqldb, mysqldialect.New())

	t.Cleanup(func() {
		_ = sqldb.Close()
	})

	return oauth.NewRepository(db), mock
}

// createState registers the INSERT expectation and creates a CSRF state.
func createState(t *testing.T, svc *oauth.Service, mock sqlmock.Sqlmock) string {
	t.Helper()

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	state, err := svc.CreateState(context.Background(), 42, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}

	return state
}

// checkMock asserts that every registered sqlmock expectation was consumed.
func checkMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// newTestService returns a service wired to the sqlmock-backed repository.
func newTestService(t *testing.T, cfg oauth.Config, refresher oauth.TokenRefresher) (*oauth.Service, sqlmock.Sqlmock) {
	t.Helper()

	svc, _, mock := newTestServiceWithRepo(t, cfg, refresher)

	return svc, mock
}

// newTestServiceWithRepo returns a service and its sqlmock-backed repository.
func newTestServiceWithRepo(
	t *testing.T, cfg oauth.Config, refresher oauth.TokenRefresher,
) (*oauth.Service, *oauth.Repository, sqlmock.Sqlmock) {
	t.Helper()

	repo, mock := newTestRepo(t)

	return oauth.NewService(cfg, repo, refresher, zap.NewNop()), repo, mock
}

// fakeRefresher counts invocations and returns the provided token.
func fakeRefresher(calls *int32, result *oauth.Token) oauth.TokenRefresher {
	return func(_ context.Context, _ string) (*oauth.Token, error) {
		atomic.AddInt32(calls, 1)
		return result, nil
	}
}

func tokenRow(token *oauth.Token) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"singleton_id", "access_token", "refresh_token", "scope",
		"expires_at", "connected_by_user_id", "created_at", "updated_at",
	}).AddRow(
		oauth.SingletonID, token.AccessToken, token.RefreshToken, token.Scope,
		token.ExpiresAt, token.ConnectedByUserID, token.CreatedAt, token.UpdatedAt,
	)
}

func hashState(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

// stateRow returns a valid stored CSRF state row for the admin user bound to
// the test redirect URI.
func stateRow(state string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
	}).AddRow(
		hashState(state), 42, testRedirectURI,
		time.Now().Add(testStateTTL), time.Now(),
	)
}

// newTestRepoUnordered returns a repository backed by sqlmock with ordered
// expectation matching disabled. Concurrent consumption tests register
// identical SELECT/DELETE expectations for every goroutine; the first DELETE
// query to arrive claims the winner expectation, the rest the losers.
func newTestRepoUnordered(t *testing.T) (*oauth.Repository, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	mock.MatchExpectationsInOrder(false)
	sqldb.SetMaxOpenConns(1)

	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version()"}).AddRow("8.0.36"))

	db := bun.NewDB(sqldb, mysqldialect.New())

	t.Cleanup(func() {
		_ = sqldb.Close()
	})

	return oauth.NewRepository(db), mock
}

func TestGetTokenReturnsValidTokenWithoutRefresh(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	now := time.Now()
	stored := &oauth.Token{
		AccessToken:       "valid-access-token",
		RefreshToken:      "valid-refresh-token",
		Scope:             "webhook repository:admin",
		ExpiresAt:         now.Add(3 * time.Hour),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	got, err := svc.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got.AccessToken != "valid-access-token" {
		t.Error("GetToken() AccessToken mismatch: token value redacted in failure message")
	}
	checkMock(t, mock)
}

func TestGetTokenRefreshesWithinThreshold(t *testing.T) {
	now := time.Now()
	refreshed := &oauth.Token{
		AccessToken:       "refreshed-access-token",
		RefreshToken:      "refreshed-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(2 * time.Hour),
		ConnectedByUserID: 42,
	}
	var calls int32
	svc, mock := newTestService(t, oauth.Config{}, fakeRefresher(&calls, refreshed))

	// Token expires in 10 minutes: inside the 15-minute proactive threshold.
	stored := &oauth.Token{
		AccessToken:       "expiring-access-token",
		RefreshToken:      "expiring-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(10 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-2 * time.Hour),
		UpdatedAt:         now.Add(-2 * time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if calls != 1 {
		t.Errorf("refresher calls = %d, want 1", calls)
	}
	if got.AccessToken != "refreshed-access-token" {
		t.Error("GetToken() AccessToken mismatch: token value redacted in failure message")
	}
	checkMock(t, mock)
}

func TestGetTokenRefreshesWhenExpired(t *testing.T) {
	now := time.Now()
	refreshed := &oauth.Token{
		AccessToken:       "refreshed-access-token",
		RefreshToken:      "refreshed-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(2 * time.Hour),
		ConnectedByUserID: 42,
	}
	var calls int32
	svc, mock := newTestService(t, oauth.Config{}, fakeRefresher(&calls, refreshed))

	stored := &oauth.Token{
		AccessToken:       "expired-access-token",
		RefreshToken:      "expired-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(-30 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-3 * time.Hour),
		UpdatedAt:         now.Add(-3 * time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := svc.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if calls != 1 {
		t.Errorf("refresher calls = %d, want 1", calls)
	}
	if got.AccessToken != "refreshed-access-token" {
		t.Error("GetToken() AccessToken mismatch: token value redacted in failure message")
	}
	checkMock(t, mock)
}

func TestGetTokenWhenNotConnected(t *testing.T) {
	var calls int32
	svc, mock := newTestService(t, oauth.Config{}, fakeRefresher(&calls, nil))

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(sqlmock.NewRows([]string{"singleton_id"}))

	_, err := svc.GetToken(context.Background())
	if !errors.Is(err, oauth.ErrOAuthNotConnected) {
		t.Errorf("GetToken() error = %v, want %v", err, oauth.ErrOAuthNotConnected)
	}
	if calls != 0 {
		t.Errorf("refresher calls = %d, want 0", calls)
	}
	checkMock(t, mock)
}

func TestGetTokenRefreshNotConfigured(t *testing.T) {
	now := time.Now()
	svc, mock := newTestService(t, oauth.Config{}, nil)

	stored := &oauth.Token{
		AccessToken:       "expiring-access-token",
		RefreshToken:      "expiring-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(5 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	_, err := svc.GetToken(context.Background())
	if !errors.Is(err, oauth.ErrRefreshNotConfigured) {
		t.Errorf("GetToken() error = %v, want %v", err, oauth.ErrRefreshNotConfigured)
	}
	checkMock(t, mock)
}

func TestGetTokenWithEmptyRefreshTokenDoesNotRefresh(t *testing.T) {
	now := time.Now()
	refreshed := &oauth.Token{
		AccessToken:       "refreshed-access-token",
		RefreshToken:      "refreshed-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(2 * time.Hour),
		ConnectedByUserID: 42,
	}
	var calls int32
	svc, mock := newTestService(t, oauth.Config{}, fakeRefresher(&calls, refreshed))

	stored := &oauth.Token{
		AccessToken:       "expiring-access-token",
		RefreshToken:      "",
		Scope:             "webhook",
		ExpiresAt:         now.Add(5 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	_, err := svc.GetToken(context.Background())
	if !errors.Is(err, oauth.ErrTokenExpired) {
		t.Errorf("GetToken() error = %v, want %v", err, oauth.ErrTokenExpired)
	}
	if calls != 0 {
		t.Errorf("refresher calls = %d, want 0", calls)
	}
	checkMock(t, mock)
}

func TestGetTokenConcurrentRefreshRunsSingleflight(t *testing.T) {
	const workers = 8

	now := time.Now()
	refreshed := &oauth.Token{
		AccessToken:       "refreshed-access-token",
		RefreshToken:      "refreshed-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(2 * time.Hour),
		ConnectedByUserID: 42,
	}

	// The refresher blocks until every worker has joined the singleflight
	// call, making the coalescing assertion deterministic. singleflight does
	// not memoize completed calls, so late arrivals would re-run the refresh.
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls int32
	refresher := func(_ context.Context, _ string) (*oauth.Token, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(entered)
		}
		<-release
		return refreshed, nil
	}

	svc, mock := newTestService(t, oauth.Config{}, refresher)

	stored := &oauth.Token{
		AccessToken:       "expiring-access-token",
		RefreshToken:      "expiring-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(5 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}

	// Singleflight serializes all DB work: the leader selects the stale row
	// and persists the refreshed tokens; followers reuse the leader's result
	// and never touch the database.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Go(func() {
			got, err := svc.GetToken(ctx)
			if err != nil {
				errs <- err
				return
			}
			if got.AccessToken != "refreshed-access-token" {
				errs <- errors.New("got stale access token after refresh")
			}
		})
	}
	<-entered
	// Give every worker time to reach the in-flight singleflight call.
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent GetToken() error = %v", err)
	}

	if calls != 1 {
		t.Errorf("refresher calls = %d, want 1 (singleflight)", calls)
	}
	checkMock(t, mock)
}

func TestGetTokenConcurrentRefreshFailurePropagatesToAll(t *testing.T) {
	const workers = 8

	now := time.Now()
	refreshErr := errors.New("refresh token revoked")

	// Same barrier as the success case: the leader blocks in the refresher
	// until all workers joined, so the single failure is shared by everyone.
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls int32
	refresher := func(_ context.Context, _ string) (*oauth.Token, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(entered)
		}
		<-release
		return nil, refreshErr
	}

	svc, mock := newTestService(t, oauth.Config{}, refresher)

	stored := &oauth.Token{
		AccessToken:       "expiring-access-token",
		RefreshToken:      "expiring-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(5 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Go(func() {
			if _, err := svc.GetToken(ctx); !errors.Is(err, refreshErr) {
				errs <- err
			}
		})
	}
	<-entered
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent GetToken() error = %v, want refresh error", err)
	}

	if calls != 1 {
		t.Errorf("refresher calls = %d, want 1 (singleflight)", calls)
	}
	checkMock(t, mock)
}

func TestSaveTokensPersistsWithDefaultLifetime(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	token := &oauth.Token{
		AccessToken:       "new-access-token",
		RefreshToken:      "new-refresh-token",
		Scope:             "webhook repository:admin",
		ConnectedByUserID: 7,
	}
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.SaveTokens(context.Background(), token); err != nil {
		t.Fatalf("SaveTokens() error = %v", err)
	}

	wantMin := time.Now().Add(7200*time.Second - 30*time.Second)
	wantMax := time.Now().Add(7200*time.Second + 30*time.Second)
	if token.ExpiresAt.Before(wantMin) || token.ExpiresAt.After(wantMax) {
		t.Errorf("SaveTokens() ExpiresAt = %v, want within 30s of now+7200s", token.ExpiresAt)
	}
	checkMock(t, mock)
}

func TestSaveTokensPreservesProvidedExpiry(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	expiresAt := time.Now().Add(45 * time.Minute)
	token := &oauth.Token{
		AccessToken:       "new-access-token",
		RefreshToken:      "new-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         expiresAt,
		ConnectedByUserID: 7,
	}
	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.SaveTokens(context.Background(), token); err != nil {
		t.Fatalf("SaveTokens() error = %v", err)
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Errorf("SaveTokens() ExpiresAt = %v, want %v", token.ExpiresAt, expiresAt)
	}
	checkMock(t, mock)
}

func TestSaveTokensScopeValidation(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	tests := []struct {
		name  string
		scope string
		want  error
	}{
		{name: "empty scope", scope: "", want: oauth.ErrInvalidScope},
		{name: "webhook only", scope: "webhook", want: nil},
		{name: "webhook among scopes", scope: "webhook repository:admin", want: nil},
		{name: "webhook comma separated", scope: "repository:admin,webhook", want: nil},
		{name: "missing webhook", scope: "repository:admin", want: oauth.ErrInvalidScope},
		{name: "case sensitive", scope: "Webhook", want: oauth.ErrInvalidScope},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &oauth.Token{
				AccessToken:       "access-token",
				RefreshToken:      "refresh-token",
				Scope:             tt.scope,
				ConnectedByUserID: 7,
			}
			if tt.want == nil {
				mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err := svc.SaveTokens(context.Background(), token)
			if !errors.Is(err, tt.want) {
				t.Errorf("SaveTokens() error = %v, want %v", err, tt.want)
			}
		})
	}

	checkMock(t, mock)
}

func TestDeleteTokens(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectExec("(?i)DELETE FROM `oauth_tokens`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteTokens(context.Background()); err != nil {
		t.Fatalf("DeleteTokens() error = %v", err)
	}
	checkMock(t, mock)
}

func TestCreateStateReturnsRandom32ByteState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	state, err := svc.CreateState(context.Background(), 42, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		t.Fatalf("state is not valid base64url: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("state length = %d bytes, want 32", len(raw))
	}
	checkMock(t, mock)
}

func TestCreateStateStoresHashedWithTTL(t *testing.T) {
	svc, repo, mock := newTestServiceWithRepo(t, oauth.Config{}, nil)

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	state, err := svc.CreateState(context.Background(), 42, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}

	// The stored row must contain the SHA-256 hash, never the plaintext state.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))

	stored, err := repo.GetState(context.Background(), hashState(state))
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}
	if stored.StateHash == state {
		t.Error("stored state hash equals plaintext state")
	}
	if stored.StateHash != hashState(state) {
		t.Errorf("stored state hash = %q, want sha256(state)", stored.StateHash)
	}
	wantMin := time.Now().Add(testStateTTL - 30*time.Second)
	wantMax := time.Now().Add(testStateTTL + 30*time.Second)
	if stored.ExpiresAt.Before(wantMin) || stored.ExpiresAt.After(wantMax) {
		t.Errorf("stored ExpiresAt = %v, want within 30s of now+10m", stored.ExpiresAt)
	}
	checkMock(t, mock)
}

func TestCreateStateProducesUniqueStates(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	first, err := svc.CreateState(context.Background(), 42, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}
	second, err := svc.CreateState(context.Background(), 42, testRedirectURI)
	if err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}
	if first == second {
		t.Error("two CreateState() calls returned identical states")
	}
	checkMock(t, mock)
}

func TestConsumeStateValidSingleUse(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	// Consumption: lookup by hash, then delete.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.ConsumeState(context.Background(), state, 42, testRedirectURI); err != nil {
		t.Fatalf("ConsumeState() error = %v", err)
	}

	// Second use must fail: the state was deleted on first use.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	err := svc.ConsumeState(context.Background(), state, 42, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("second ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateRejectsWrongUser(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	// The row is only read: the user binding is checked before the
	// conditional delete, so the rejected attempt must not burn the state.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))

	err := svc.ConsumeState(context.Background(), state, 999, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}

	// The bound admin can still consume the state afterwards.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if consumeErr := svc.ConsumeState(context.Background(), state, 42, testRedirectURI); consumeErr != nil {
		t.Fatalf("ConsumeState() after rejected attempt error = %v", consumeErr)
	}
	checkMock(t, mock)
}

func TestConsumeStateRejectsWrongRedirectURI(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	// The row is only read: the redirect URI binding is checked before the
	// conditional delete, so the rejected attempt must not burn the state.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))

	err := svc.ConsumeState(context.Background(), state, 42, "https://evil.example.com")
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}

	// The state bound to the registered redirect URI is still consumable.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if consumeErr := svc.ConsumeState(context.Background(), state, 42, testRedirectURI); consumeErr != nil {
		t.Fatalf("ConsumeState() after rejected attempt error = %v", consumeErr)
	}
	checkMock(t, mock)
}

func TestConsumeStateRejectsExpiredState(t *testing.T) {
	// Negative TTL puts every created state in the past.
	svc, mock := newTestService(t, oauth.Config{StateTTL: -time.Minute}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(-time.Minute), time.Now(),
		))

	err := svc.ConsumeState(context.Background(), state, 42, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateRejectsUnknownState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	err := svc.ConsumeState(context.Background(), "unknown-state", 42, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateRejectsEmptyState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	err := svc.ConsumeState(context.Background(), "", 42, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateConcurrentSingleWinner(t *testing.T) {
	const workers = 8

	repo, mock := newTestRepoUnordered(t)
	svc := oauth.NewService(oauth.Config{}, repo, nil, zap.NewNop())

	state := createState(t, svc, mock)

	// Every goroutine reads the row; the conditional DELETE is the atomic
	// single-use gate: exactly one DELETE deletes one row, the rest none.
	for range workers {
		mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
			WillReturnRows(stateRow(state))
	}
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for range workers - 1 {
		mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	ctx := context.Background()
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			if err := svc.ConsumeState(ctx, state, 42, testRedirectURI); err != nil {
				results <- err
			}
		})
	}
	wg.Wait()
	close(results)

	notFound := 0
	for err := range results {
		if !errors.Is(err, oauth.ErrStateNotFound) {
			t.Errorf("concurrent ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
			continue
		}
		notFound++
	}
	if successes := workers - notFound; successes != 1 {
		t.Errorf("successful consumptions = %d, want exactly 1", successes)
	}
	checkMock(t, mock)
}

func TestConsumeStateDeleteFailureRejects(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	delErr := errors.New("database unavailable")
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnError(delErr)

	err := svc.ConsumeState(context.Background(), state, 42, testRedirectURI)
	if !errors.Is(err, delErr) {
		t.Errorf("ConsumeState() error = %v, want wrap of %v", err, delErr)
	}
	if strings.Contains(err.Error(), state) || strings.Contains(err.Error(), hashState(state)) {
		t.Error("ConsumeState() error message leaks the state or its hash")
	}
	checkMock(t, mock)
}

func TestConsumeStateDeleteZeroRowsRejects(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.ConsumeState(context.Background(), state, 42, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestValidateScope(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	tests := []struct {
		scope string
		want  bool
	}{
		{scope: "", want: false},
		{scope: "webhook", want: true},
		{scope: "webhook repository:admin", want: true},
		{scope: "repository:admin webhook", want: true},
		{scope: "repository:admin", want: false},
		{scope: "notwebhook", want: false},
		{scope: "Webhook", want: false},
	}
	for _, tt := range tests {
		err := svc.ValidateScope(tt.scope)
		if tt.want && err != nil {
			t.Errorf("ValidateScope(%q) error = %v, want nil", tt.scope, err)
		}
		if !tt.want && !errors.Is(err, oauth.ErrInvalidScope) {
			t.Errorf("ValidateScope(%q) error = %v, want %v", tt.scope, err, oauth.ErrInvalidScope)
		}
	}
	checkMock(t, mock)
}

func TestErrorMessagesDoNotContainTokens(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	secret := "super-secret-access-token-value"
	stored := &oauth.Token{
		AccessToken:       secret,
		RefreshToken:      "super-secret-refresh-token-value",
		Scope:             "webhook",
		ExpiresAt:         time.Now().Add(5 * time.Minute),
		ConnectedByUserID: 42,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	_, err := svc.GetToken(context.Background())
	if err == nil {
		t.Fatal("GetToken() error = nil, want ErrRefreshNotConfigured")
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("error message leaks the access token")
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeReturnsBoundUserID(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	userID, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI)
	if err != nil {
		t.Fatalf("ConsumeStateForExchange() error = %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeSingleUse(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI); err != nil {
		t.Fatalf("ConsumeStateForExchange() error = %v", err)
	}

	// Second use must fail: the state was deleted on first use.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	_, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("second ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeRejectsWrongRedirectURI(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	// The row is only read: the redirect URI binding is checked before the
	// conditional delete, so the rejected attempt must not burn the state.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(testStateTTL), time.Now(),
		))

	_, err := svc.ConsumeStateForExchange(context.Background(), state, "https://evil.example.com")
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}

	// The state bound to the registered redirect URI is still consumable.
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, consumeErr := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI); consumeErr != nil {
		t.Fatalf("ConsumeStateForExchange() after rejected attempt error = %v", consumeErr)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeRejectsExpiredState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{StateTTL: -time.Minute}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			hashState(state), 42, testRedirectURI,
			time.Now().Add(-time.Minute), time.Now(),
		))

	_, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeRejectsUnknownState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	_, err := svc.ConsumeStateForExchange(context.Background(), "unknown-state", testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeRejectsEmptyState(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	_, err := svc.ConsumeStateForExchange(context.Background(), "", testRedirectURI)
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeConcurrentSingleWinner(t *testing.T) {
	const workers = 8

	repo, mock := newTestRepoUnordered(t)
	svc := oauth.NewService(oauth.Config{}, repo, nil, zap.NewNop())

	state := createState(t, svc, mock)

	// Every goroutine reads the row; the conditional DELETE is the atomic
	// single-use gate: exactly one DELETE deletes one row, the rest none.
	for range workers {
		mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
			WillReturnRows(stateRow(state))
	}
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for range workers - 1 {
		mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	ctx := context.Background()
	results := make(chan error, workers)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			userID, err := svc.ConsumeStateForExchange(ctx, state, testRedirectURI)
			if err != nil {
				results <- err
				return
			}
			if userID != 42 {
				results <- fmt.Errorf("userID = %d, want 42", userID)
				return
			}
			successes.Add(1)
		})
	}
	wg.Wait()
	close(results)

	for err := range results {
		if !errors.Is(err, oauth.ErrStateNotFound) {
			t.Errorf("concurrent ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
		}
	}
	if successes.Load() != 1 {
		t.Errorf("successful consumptions = %d, want exactly 1", successes.Load())
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeDeleteFailureRejects(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	delErr := errors.New("database unavailable")
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnError(delErr)

	userID, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI)
	if userID != 0 {
		t.Errorf("userID = %d, want 0 on delete failure", userID)
	}
	if !errors.Is(err, delErr) {
		t.Errorf("ConsumeStateForExchange() error = %v, want wrap of %v", err, delErr)
	}
	if strings.Contains(err.Error(), state) || strings.Contains(err.Error(), hashState(state)) {
		t.Error("ConsumeStateForExchange() error message leaks the state or its hash")
	}
	checkMock(t, mock)
}

func TestConsumeStateForExchangeDeleteZeroRowsRejects(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	state := createState(t, svc, mock)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(stateRow(state))
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 0))

	userID, err := svc.ConsumeStateForExchange(context.Background(), state, testRedirectURI)
	if userID != 0 {
		t.Errorf("userID = %d, want 0 when the state is already consumed", userID)
	}
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("ConsumeStateForExchange() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestGetStoredTokenReturnsRowWithoutRefresh(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	now := time.Now()
	stored := &oauth.Token{
		AccessToken:       "valid-access-token",
		RefreshToken:      "valid-refresh-token",
		Scope:             "webhook",
		ExpiresAt:         now.Add(3 * time.Hour),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}
	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(stored))

	got, err := svc.GetStoredToken(context.Background())
	if err != nil {
		t.Fatalf("GetStoredToken() error = %v", err)
	}
	if got.AccessToken != "valid-access-token" {
		t.Error("GetStoredToken() AccessToken mismatch: token value redacted in failure message")
	}
	checkMock(t, mock)
}

func TestGetStoredTokenWhenNotConnected(t *testing.T) {
	svc, mock := newTestService(t, oauth.Config{}, nil)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(sqlmock.NewRows([]string{"singleton_id"}))

	_, err := svc.GetStoredToken(context.Background())
	if !errors.Is(err, oauth.ErrOAuthNotConnected) {
		t.Errorf("GetStoredToken() error = %v, want %v", err, oauth.ErrOAuthNotConnected)
	}
	checkMock(t, mock)
}
