package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// RequiredScope is the minimum Bitbucket OAuth scope for webhook management.
const RequiredScope = "webhook"

// stateSize is the byte length of the cryptographically random CSRF state.
const stateSize = 32

// TokenRefresher exchanges a refresh token for a fresh access token. The
// refresh token is single-use in Bitbucket Cloud; every response carries a new
// one. Implementations must never log or serialize raw tokens.
type TokenRefresher func(ctx context.Context, refreshToken string) (*Token, error)

// singleflightKey serializes every refresh attempt, including the read and
// persist of the singleton row, so concurrent GetToken calls share one result.
const singleflightKey = "oauth-token-refresh"

// Service stores the singleton Bitbucket OAuth credential and manages CSRF
// states for the registration flow. Tokens are stored plaintext for the MVP.
type Service struct {
	cfg       Config
	repo      *Repository
	refresher TokenRefresher
	group     singleflight.Group
	logger    *zap.Logger
}

func NewService(cfg Config, repo *Repository, refresher TokenRefresher, logger *zap.Logger) *Service {
	return &Service{
		cfg:       normalizeConfig(cfg),
		repo:      repo,
		refresher: refresher,
		group:     singleflight.Group{},
		logger:    logger,
	}
}

// SetRefresher wires the token exchange used by GetToken. It is a separate
// setter because the exchange client is provided by another module.
func (s *Service) SetRefresher(refresher TokenRefresher) {
	s.refresher = refresher
}

// SaveTokens upserts the singleton token row after validating the scope.
// A zero ExpiresAt is replaced with now + AccessTokenLifetime (7200s).
func (s *Service) SaveTokens(ctx context.Context, token *Token) error {
	if err := s.ValidateScope(token.Scope); err != nil {
		return err
	}

	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().Add(s.cfg.AccessTokenLifetime)
	}

	if err := s.repo.SaveTokens(ctx, token); err != nil {
		return fmt.Errorf("failed to save oauth tokens: %w", err)
	}

	s.logger.Info("oauth tokens saved", zap.Int64("user_id", token.ConnectedByUserID))

	return nil
}

// GetToken returns the current access token. When the token is within
// RefreshThreshold of expiry (or already expired) it is refreshed first.
// Concurrent calls share a single refresh via singleflight.
func (s *Service) GetToken(ctx context.Context) (*Token, error) {
	value, err, _ := s.group.Do(singleflightKey, func() (any, error) {
		token, getErr := s.repo.GetToken(ctx)
		if getErr != nil {
			return nil, getErr
		}

		if time.Now().Add(s.cfg.RefreshThreshold).Before(token.ExpiresAt) {
			return token, nil
		}

		refreshed, refreshErr := s.refreshLocked(ctx, token)
		if refreshErr != nil {
			return nil, refreshErr
		}

		return refreshed, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth token: %w", err)
	}

	token, ok := value.(*Token)
	if !ok {
		return nil, errUnexpectedTokenType
	}

	return token, nil
}

// refreshLocked runs a single refresh cycle. Callers must hold the
// singleflight lock (GetToken wraps this call).
func (s *Service) refreshLocked(ctx context.Context, current *Token) (*Token, error) {
	if s.refresher == nil {
		return nil, ErrRefreshNotConfigured
	}
	if current.RefreshToken == "" {
		return nil, ErrTokenExpired
	}

	s.logger.Info("refreshing oauth access token")

	refreshed, err := s.refresher(ctx, current.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh oauth token: %w", err)
	}

	if scopeErr := s.ValidateScope(refreshed.Scope); scopeErr != nil {
		return nil, scopeErr
	}

	if saveErr := s.repo.SaveTokens(ctx, refreshed); saveErr != nil {
		return nil, fmt.Errorf("failed to persist refreshed oauth tokens: %w", saveErr)
	}

	return refreshed, nil
}

// DeleteTokens removes the singleton token row. Deleting when not connected
// is a no-op so disconnect stays idempotent.
func (s *Service) DeleteTokens(ctx context.Context) error {
	if err := s.repo.DeleteTokens(ctx); err != nil {
		return fmt.Errorf("failed to delete oauth tokens: %w", err)
	}

	s.logger.Info("oauth tokens deleted")

	return nil
}

// CreateState generates a cryptographically random 32-byte CSRF state bound
// to the admin user and redirect URI. Only the SHA-256 hash is stored; the
// plaintext state is returned once for the authorization redirect.
func (s *Service) CreateState(ctx context.Context, userID int64, redirectURI string) (string, error) {
	raw := make([]byte, stateSize)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}

	state := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()

	if err := s.repo.CreateState(ctx, &State{
		StateHash:   hashState(state),
		UserID:      userID,
		RedirectURI: redirectURI,
		ExpiresAt:   now.Add(s.cfg.StateTTL),
		CreatedAt:   now,
	}); err != nil {
		return "", fmt.Errorf("failed to store oauth state: %w", err)
	}

	return state, nil
}

// ConsumeState verifies a single-use CSRF state in constant time against its
// stored hash, checks the user binding, redirect URI, and TTL, then consumes
// the state with a conditional delete. Validation happens before deletion so
// an invalid attempt never burns a valid state; the conditional delete is the
// atomic single-use gate: the state hash is the primary key, so concurrent
// attempts share at most one winner. A failed deletion rejects the attempt
// instead of continuing.
func (s *Service) ConsumeState(ctx context.Context, state string, userID int64, redirectURI string) error {
	if state == "" {
		return ErrStateNotFound
	}

	presented := hashState(state)

	stored, err := s.repo.GetState(ctx, presented)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(stored.StateHash), []byte(presented)) != 1 {
		return ErrStateNotFound
	}
	if stored.UserID != userID {
		return ErrStateNotFound
	}
	if stored.RedirectURI != redirectURI {
		return ErrStateNotFound
	}
	if time.Now().After(stored.ExpiresAt) {
		return ErrStateNotFound
	}

	deleted, delErr := s.repo.DeleteState(ctx, stored.StateHash)
	if delErr != nil {
		return fmt.Errorf("failed to consume oauth state: %w", delErr)
	}
	if !deleted {
		return ErrStateNotFound
	}

	return nil
}

// ConsumeStateForExchange verifies and consumes the single-use CSRF state
// presented on the public OAuth callback. The callback has no authenticated
// principal, so the user binding stored at creation time is verified against
// the state row itself and returned for attributing the connection. The
// redirect URI and expiry are validated before the conditional delete, which
// is the atomic single-use gate: the state hash is the primary key, so
// concurrent callbacks share at most one winner. A failed deletion rejects
// the attempt, so the authorization code is never exchanged.
func (s *Service) ConsumeStateForExchange(ctx context.Context, state string, redirectURI string) (int64, error) {
	if state == "" {
		return 0, ErrStateNotFound
	}

	presented := hashState(state)

	stored, err := s.repo.GetState(ctx, presented)
	if err != nil {
		return 0, err
	}

	if subtle.ConstantTimeCompare([]byte(stored.StateHash), []byte(presented)) != 1 {
		return 0, ErrStateNotFound
	}
	if stored.RedirectURI != redirectURI {
		return 0, ErrStateNotFound
	}
	if time.Now().After(stored.ExpiresAt) {
		return 0, ErrStateNotFound
	}

	deleted, delErr := s.repo.DeleteState(ctx, stored.StateHash)
	if delErr != nil {
		return 0, fmt.Errorf("failed to consume oauth state: %w", delErr)
	}
	if !deleted {
		return 0, ErrStateNotFound
	}

	return stored.UserID, nil
}

// GetStoredToken returns the persisted OAuth credential without triggering a
// refresh. It backs the connection status endpoint, which must be free of
// refresh side effects.
func (s *Service) GetStoredToken(ctx context.Context) (*Token, error) {
	token, err := s.repo.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth token: %w", err)
	}

	return token, nil
}

// ValidateScope requires the 'webhook' scope in a space- or comma-separated
// scope string as returned by the Bitbucket OAuth exchange.
func (s *Service) ValidateScope(scope string) error {
	if scope == "" {
		return ErrInvalidScope
	}

	parts := strings.FieldsFunc(scope, func(r rune) bool {
		return r == ' ' || r == ','
	})
	if slices.Contains(parts, RequiredScope) {
		return nil
	}

	return ErrInvalidScope
}

// hashState returns the hex SHA-256 digest used as the storage key.
func hashState(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}
