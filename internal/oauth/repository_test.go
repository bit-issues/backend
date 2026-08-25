package oauth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bit-issues/backend/internal/oauth"
)

func TestRepositorySaveTokensUpsertsSingletonRow(t *testing.T) {
	repo, mock := newTestRepo(t)

	token := &oauth.Token{
		AccessToken:       "access-token",
		RefreshToken:      "refresh-token",
		Scope:             "webhook",
		ExpiresAt:         time.Now().Add(time.Hour),
		ConnectedByUserID: 42,
	}

	mock.ExpectExec("(?i)INSERT INTO `oauth_tokens`.*ON DUPLICATE KEY UPDATE").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.SaveTokens(context.Background(), token); err != nil {
		t.Fatalf("SaveTokens() error = %v", err)
	}
	checkMock(t, mock)
}

func TestRepositoryGetTokenReturnsRow(t *testing.T) {
	repo, mock := newTestRepo(t)

	now := time.Now()
	token := &oauth.Token{
		AccessToken:       "access-token",
		RefreshToken:      "refresh-token",
		Scope:             "webhook repository:admin",
		ExpiresAt:         now.Add(time.Hour),
		ConnectedByUserID: 42,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-time.Hour),
	}

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(tokenRow(token))

	got, err := repo.GetToken(context.Background())
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got.AccessToken != token.AccessToken {
		t.Error("GetToken() AccessToken mismatch: token value redacted in failure message")
	}
	if got.RefreshToken != token.RefreshToken {
		t.Error("GetToken() RefreshToken mismatch: token value redacted in failure message")
	}
	if got.Scope != token.Scope {
		t.Errorf("GetToken() Scope = %q, want %q", got.Scope, token.Scope)
	}
	if !got.ExpiresAt.Equal(token.ExpiresAt) {
		t.Errorf("GetToken() ExpiresAt = %v, want %v", got.ExpiresAt, token.ExpiresAt)
	}
	if got.ConnectedByUserID != token.ConnectedByUserID {
		t.Errorf("GetToken() ConnectedByUserID = %d, want %d", got.ConnectedByUserID, token.ConnectedByUserID)
	}
	checkMock(t, mock)
}

func TestRepositoryGetTokenReturnsNotConnectedWhenEmpty(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_tokens`").
		WillReturnRows(sqlmock.NewRows([]string{"singleton_id"}))

	_, err := repo.GetToken(context.Background())
	if !errors.Is(err, oauth.ErrOAuthNotConnected) {
		t.Errorf("GetToken() error = %v, want %v", err, oauth.ErrOAuthNotConnected)
	}
	checkMock(t, mock)
}

func TestRepositoryDeleteTokens(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectExec("(?i)DELETE FROM `oauth_tokens`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.DeleteTokens(context.Background()); err != nil {
		t.Fatalf("DeleteTokens() error = %v", err)
	}
	checkMock(t, mock)
}

func TestRepositoryCreateState(t *testing.T) {
	repo, mock := newTestRepo(t)

	state := &oauth.State{
		StateHash:   "abc123",
		UserID:      42,
		RedirectURI: testRedirectURI,
		ExpiresAt:   time.Now().Add(testStateTTL),
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("(?i)INSERT INTO `oauth_states`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.CreateState(context.Background(), state); err != nil {
		t.Fatalf("CreateState() error = %v", err)
	}
	checkMock(t, mock)
}

func TestRepositoryGetStateReturnsRow(t *testing.T) {
	repo, mock := newTestRepo(t)

	now := time.Now()
	state := &oauth.State{
		StateHash:   "abc123",
		UserID:      42,
		RedirectURI: testRedirectURI,
		ExpiresAt:   now.Add(testStateTTL),
		CreatedAt:   now,
	}

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{
			"state_hash", "user_id", "redirect_uri", "expires_at", "created_at",
		}).AddRow(
			state.StateHash, state.UserID, state.RedirectURI, state.ExpiresAt, state.CreatedAt,
		))

	got, err := repo.GetState(context.Background(), state.StateHash)
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}
	if got.StateHash != state.StateHash {
		t.Errorf("GetState() StateHash = %q, want %q", got.StateHash, state.StateHash)
	}
	if got.UserID != state.UserID {
		t.Errorf("GetState() UserID = %d, want %d", got.UserID, state.UserID)
	}
	if got.RedirectURI != state.RedirectURI {
		t.Errorf("GetState() RedirectURI = %q, want %q", got.RedirectURI, state.RedirectURI)
	}
	if !got.ExpiresAt.Equal(state.ExpiresAt) {
		t.Errorf("GetState() ExpiresAt = %v, want %v", got.ExpiresAt, state.ExpiresAt)
	}
	checkMock(t, mock)
}

func TestRepositoryGetStateReturnsNotFoundWhenEmpty(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectQuery("(?i)SELECT .* FROM `oauth_states`").
		WillReturnRows(sqlmock.NewRows([]string{"state_hash"}))

	_, err := repo.GetState(context.Background(), "missing")
	if !errors.Is(err, oauth.ErrStateNotFound) {
		t.Errorf("GetState() error = %v, want %v", err, oauth.ErrStateNotFound)
	}
	checkMock(t, mock)
}

func TestRepositoryDeleteState(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleted, err := repo.DeleteState(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("DeleteState() error = %v", err)
	}
	if !deleted {
		t.Error("DeleteState() deleted = false, want true for an existing row")
	}
	checkMock(t, mock)
}

func TestRepositoryDeleteStateMissingRowReturnsFalse(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := repo.DeleteState(context.Background(), "missing")
	if err != nil {
		t.Fatalf("DeleteState() error = %v", err)
	}
	if deleted {
		t.Error("DeleteState() deleted = true, want false for a missing row")
	}
	checkMock(t, mock)
}

func TestRepositoryDeleteStateErrorPropagates(t *testing.T) {
	repo, mock := newTestRepo(t)

	delErr := errors.New("database unavailable")
	mock.ExpectExec("(?i)DELETE FROM `oauth_states`").
		WillReturnError(delErr)

	deleted, err := repo.DeleteState(context.Background(), "abc123")
	if !errors.Is(err, delErr) {
		t.Errorf("DeleteState() error = %v, want wrap of %v", err, delErr)
	}
	if deleted {
		t.Error("DeleteState() deleted = true, want false on error")
	}
	checkMock(t, mock)
}
