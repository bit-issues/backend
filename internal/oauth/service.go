package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// singleflightKey serializes every refresh attempt, including the read and
// persist of the singleton row, so concurrent GetToken calls share one result.
const singleflightKey = "oauth-token-refresh"

// Service stores the singleton Bitbucket OAuth credential and manages the registration flow.
type Service struct {
	cfg    Config
	tokens *Repository

	group singleflight.Group
	http  *http.Client

	logger *zap.Logger
}

func NewService(
	cfg Config,
	tokens *Repository,

	logger *zap.Logger,
) *Service {
	return &Service{
		cfg:    cfg,
		tokens: tokens,

		group: singleflight.Group{},
		http:  http.DefaultClient,

		logger: logger,
	}
}

func (s *Service) AuthorizeURL(_ context.Context, userID int64) string {
	query := url.Values{}
	query.Set("client_id", s.cfg.ClientID)
	query.Set("response_type", "code")
	query.Set("scope", requiredScope)
	query.Set("state", strconv.FormatInt(userID, 16))

	return defaultAuthorizeURL + "?" + query.Encode()
}

func (s *Service) Exchange(ctx context.Context, state, code string) error {
	userID, err := s.verifyState(ctx, state)
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)

	token, err := s.requestToken(ctx, form)
	if err != nil {
		return err
	}

	return s.tokens.Upsert(ctx, userID, token)
}

func (s *Service) GetToken(ctx context.Context, userID int64) (*Token, error) {
	value, err, _ := s.group.Do(singleflightKey+strconv.FormatInt(userID, 10), func() (any, error) {
		token, getErr := s.tokens.Get(ctx, userID)
		if getErr != nil {
			return nil, getErr
		}

		if time.Now().Add(defaultRefreshThreshold).Before(token.ExpiresAt) {
			return token, nil
		}

		refreshed, refreshErr := s.refreshLocked(ctx, userID, token)
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
		return nil, ErrTokenIssueFailed
	}

	return token, nil
}

func (s *Service) DeleteToken(ctx context.Context, userID int64) error {
	return s.tokens.Delete(ctx, userID)
}

func (s *Service) verifyState(_ context.Context, state string) (int64, error) {
	userID, err := strconv.ParseInt(state, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse state: %w", err)
	}

	return userID, nil
}

// requestToken performs an OAuth token grant. The client authenticates with
// HTTP Basic credentials. Errors never include upstream bodies or tokens.
func (s *Service) requestToken(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read oauth token response: %w", err)
	}

	var parsed tokenResponse
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse oauth token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || parsed.AccessToken == "" {
		return nil, fmt.Errorf("%w: Bitbucket returned status %d", ErrTokenIssueFailed, resp.StatusCode)
	}

	now := time.Now()
	token := &Token{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		Scopes:       parsed.Scopes,
		ExpiresAt:    time.Time{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if parsed.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	}

	return token, nil
}

// refreshLocked runs a single refresh cycle. Callers must hold the
// singleflight lock (GetToken wraps this call).
func (s *Service) refreshLocked(ctx context.Context, userID int64, current *Token) (*Token, error) {
	s.logger.Info("refreshing oauth access token")

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", current.RefreshToken)

	refreshed, err := s.requestToken(ctx, form)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh oauth token: %w", err)
	}

	return refreshed, s.tokens.Upsert(ctx, userID, refreshed)
}
