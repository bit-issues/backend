package management

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"go.uber.org/zap"
)

const (
	// webhookPath is appended to the configured callback base URL to form the
	// webhook delivery URL registered in Bitbucket.
	webhookPath = "/api/v1/webhooks/bitbucket/push"
	// hookDescription is the human-readable label of managed hooks.
	hookDescription = "BitIssues push webhook"
)

// Service manages Bitbucket repository webhooks for projects, gated on the
// requesting admin's OAuth token. It is fully stateless.
type Service struct {
	config Config

	client      *bitbucket.Client
	oauthSvc    *oauth.Service
	projectsSvc *projects.Service
	logger      *zap.Logger
}

// NewService wires the management service. The Bitbucket client is the leaf
// pkg/bitbucket client; the OAuth service resolves the requesting admin's
// token; the projects service resolves slugs to repositories.
func NewService(
	config Config,
	client *bitbucket.Client,
	oauthSvc *oauth.Service,
	projectsSvc *projects.Service,
	logger *zap.Logger,
) *Service {
	return &Service{
		config:      config,
		client:      client,
		oauthSvc:    oauthSvc,
		projectsSvc: projectsSvc,
		logger:      logger,
	}
}

// Status reports the live Bitbucket webhook state for a project. Local
// prerequisites (callback URL, OAuth connection) are reported as an
// unavailable status instead of errors; a missing secret never blocks status.
func (s *Service) Status(ctx context.Context, userID int64, projectSlug string) (*WebhookStatus, error) {
	webhookURL, err := s.resolveCallbackURL()
	if err != nil {
		return s.unavailable(err), nil
	}

	token, err := s.getToken(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrOAuthDisabled) || errors.Is(err, ErrOAuthNotConnected) {
			return s.unavailable(err), nil
		}
		return nil, err
	}

	repo, err := s.resolveRepo(ctx, projectSlug)
	if err != nil {
		return nil, err
	}

	hooks, err := s.client.ListHooks(ctx, repo.workspace, repo.slug, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list bitbucket hooks: %w", err)
	}

	return webhookStatusFromHook(findMatchingHook(hooks, webhookURL), webhookURL), nil
}

// Register creates the project webhook in Bitbucket, or updates the existing
// one when a hook already targets our callback URL. Registration fails closed
// on an empty secret or invalid callback URL.
func (s *Service) Register(ctx context.Context, userID int64, projectSlug string) (*WebhookStatus, error) {
	webhookURL, err := s.resolveCallbackURL()
	if err != nil {
		return nil, err
	}

	if s.config.Secret == "" {
		return nil, ErrSecretNotConfigured
	}

	token, err := s.getToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	repo, err := s.resolveRepo(ctx, projectSlug)
	if err != nil {
		return nil, err
	}

	hooks, err := s.client.ListHooks(ctx, repo.workspace, repo.slug, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list bitbucket hooks: %w", err)
	}

	input := bitbucket.HookInput{
		Description: hookDescription,
		URL:         webhookURL,
		Active:      true,
		Secret:      s.config.Secret,
		Events:      []bitbucket.Event{bitbucket.EventRepoPush},
	}

	var hook *bitbucket.Hook
	if existing := findMatchingHook(hooks, webhookURL); existing != nil {
		hook, err = s.client.UpdateHook(ctx, repo.workspace, repo.slug, existing.UUID, token.AccessToken, input)
		if err != nil {
			return nil, fmt.Errorf("failed to update bitbucket hook: %w", err)
		}
	} else {
		hook, err = s.client.CreateHook(ctx, repo.workspace, repo.slug, token.AccessToken, input)
		if err != nil {
			return nil, fmt.Errorf("failed to create bitbucket hook: %w", err)
		}
	}

	return webhookStatusFromHook(hook, webhookURL), nil
}

// Unregister deletes the project webhook from Bitbucket. Deleting a missing
// hook is an idempotent no-op.
func (s *Service) Unregister(ctx context.Context, userID int64, projectSlug string) error {
	webhookURL, err := s.resolveCallbackURL()
	if err != nil {
		return err
	}

	token, err := s.getToken(ctx, userID)
	if err != nil {
		return err
	}

	repo, err := s.resolveRepo(ctx, projectSlug)
	if err != nil {
		return err
	}

	hooks, err := s.client.ListHooks(ctx, repo.workspace, repo.slug, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to list bitbucket hooks: %w", err)
	}

	existing := findMatchingHook(hooks, webhookURL)
	if existing == nil {
		return nil
	}

	err = s.client.DeleteHook(ctx, repo.workspace, repo.slug, existing.UUID, token.AccessToken)
	if err != nil {
		if errors.Is(err, bitbucket.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("failed to delete bitbucket hook: %w", err)
	}

	return nil
}

// resolveCallbackURL validates the configured callback base URL and returns
// the full webhook delivery URL.
func (s *Service) resolveCallbackURL() (string, error) {
	return validateCallbackURL(s.config.CallbackURL)
}

// validateCallbackURL validates a callback base URL: https only, no query
// string or fragment, host required. Empty input is a configuration gap, not
// an invalid value.
func validateCallbackURL(raw string) (string, error) {
	base := strings.TrimRight(raw, "/")
	if base == "" {
		return "", ErrCallbackURLNotConfigured
	}

	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", ErrCallbackURLInvalid
	}

	return base + webhookPath, nil
}

// getToken resolves the requesting admin's Bitbucket OAuth token and maps
// OAuth failures to domain sentinels.
func (s *Service) getToken(ctx context.Context, userID int64) (*oauth.Token, error) {
	token, err := s.oauthSvc.GetToken(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, oauth.ErrDisabled):
			return nil, ErrOAuthDisabled
		case errors.Is(err, oauth.ErrNotFound), errors.Is(err, oauth.ErrTokenIssueFailed):
			return nil, ErrOAuthNotConnected
		default:
			return nil, fmt.Errorf("failed to get oauth token: %w", err)
		}
	}

	return token, nil
}

// resolveRepo loads the project and parses its repository URL into a
// Bitbucket workspace and slug.
func (s *Service) resolveRepo(ctx context.Context, projectSlug string) (repoRef, error) {
	project, err := s.projectsSvc.GetBySlug(ctx, projectSlug)
	if err != nil {
		return repoRef{}, fmt.Errorf("failed to get project: %w", err)
	}

	workspace, slug, err := bitbucket.ParseRepoSlug(project.RepoURL)
	if err != nil {
		return repoRef{}, fmt.Errorf("%w: %s", ErrNotBitbucketRepo, project.RepoURL)
	}

	return repoRef{workspace: workspace, slug: slug}, nil
}

// unavailable builds the unavailable status for a local prerequisite error.
func (s *Service) unavailable(err error) *WebhookStatus {
	reason := "Bitbucket webhook management is unavailable"
	switch {
	case errors.Is(err, ErrCallbackURLNotConfigured):
		reason = "Bitbucket callback URL is not configured"
	case errors.Is(err, ErrCallbackURLInvalid):
		reason = "Bitbucket callback URL is invalid"
	case errors.Is(err, ErrOAuthDisabled):
		reason = "Bitbucket OAuth is disabled"
	case errors.Is(err, ErrOAuthNotConnected):
		reason = "Bitbucket OAuth is not connected"
	}

	return &WebhookStatus{
		Status:        StatusUnavailable,
		FailureReason: &reason,
		HookUUID:      nil,
		CallbackURL:   "",
		UpdatedAt:     nil,
	}
}

// findMatchingHook returns the hook whose URL equals our webhook URL and
// which subscribes to the managed push event, or nil. A hook on the same URL
// without that event is not the managed hook and is left untouched.
func findMatchingHook(hooks []bitbucket.Hook, webhookURL string) *bitbucket.Hook {
	for i := range hooks {
		if hooks[i].URL != webhookURL {
			continue
		}
		if slices.Contains(hooks[i].Events, bitbucket.EventRepoPush) {
			return &hooks[i]
		}
	}
	return nil
}

// webhookStatusFromHook derives the canonical status from a matched hook. A
// nil hook means no matching hook is registered.
func webhookStatusFromHook(hook *bitbucket.Hook, webhookURL string) *WebhookStatus {
	status := &WebhookStatus{
		Status:        StatusNotRegistered,
		CallbackURL:   webhookURL,
		HookUUID:      nil,
		FailureReason: nil,
		UpdatedAt:     nil,
	}
	if hook == nil {
		return status
	}

	uuid := hook.UUID
	status.HookUUID = &uuid
	if hook.Active {
		status.Status = StatusRegistered
	} else {
		status.Status = StatusFailed
	}
	if !hook.UpdatedOn.IsZero() {
		updated := hook.UpdatedOn.Format(time.RFC3339)
		status.UpdatedAt = &updated
	}

	return status
}
