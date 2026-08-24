package webhooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/pkg/bitbucket"
)

const (
	// WebhookStatusRegistered marks a project whose Bitbucket repository has
	// a webhook pointing at the configured callback URL.
	WebhookStatusRegistered = "registered"

	// WebhookStatusNotRegistered marks a project without such a webhook.
	WebhookStatusNotRegistered = "not_registered"
)

const (
	// webhookCallbackPath is the receiver route webhook events are delivered
	// to: /api/v1/webhooks/bitbucket/push (registered by the server module).
	webhookCallbackPath = "/api/v1/webhooks/bitbucket/push"

	// webhookEvent is the Bitbucket event the registered webhook listens to.
	webhookEvent = "repo:push"

	// webhookDescription is used for newly created webhooks.
	webhookDescription = "BitIssues webhook"
)

// WebhookStatus describes the live Bitbucket webhook registration state of a
// project. It is derived from Bitbucket on every call and never persisted.
type WebhookStatus struct {
	Status      string
	HookUUID    string
	CallbackURL string
	UpdatedAt   string
}

// ManagementService manages the Bitbucket webhook registration of project
// repositories. All state is derived live from the Bitbucket API; nothing is
// persisted to the database.
type ManagementService struct {
	config Config

	// callbackBaseURL is BITBUCKET__CALLBACK_URL without trailing slashes.
	callbackBaseURL string
	client          *bitbucket.Client
}

// NewManagementService creates the webhook management service. The callback
// URL base is taken from the Bitbucket config; the webhook secret comes from
// the webhooks config.
func NewManagementService(
	cfg Config,
	bitbucketCfg bitbucket.Config,
	client *bitbucket.Client,
) *ManagementService {
	return &ManagementService{
		config:          cfg,
		callbackBaseURL: strings.TrimRight(bitbucketCfg.CallbackURL, "/"),
		client:          client,
	}
}

// RegisterWebhook ensures the project repository has a Bitbucket webhook
// pointing at the configured callback URL. Existing hooks are updated in
// place to enforce active state, push events, and the configured secret;
// missing hooks are created. Webhook state is never written to the database.
// Returns ErrWebhookSecretNotConfigured when WEBHOOKS__SECRET is empty.
func (s *ManagementService) RegisterWebhook(ctx context.Context, project *projects.Project) (*WebhookStatus, error) {
	if s.config.Secret == "" {
		return nil, fmt.Errorf("%w: webhook secret is not configured", ErrWebhookSecretNotConfigured)
	}

	workspace, repoSlug, err := projects.ParseBitbucketRepoURL(project.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	callbackURL := s.webhookCallbackURL()
	hooks, err := s.client.ListWebhooks(ctx, workspace, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	if hook := findWebhookByURL(hooks, callbackURL); hook != nil {
		updated, updateErr := s.client.UpdateWebhook(
			ctx,
			workspace,
			repoSlug,
			hook.UUID,
			bitbucket.UpdateWebhookRequest{
				Description: hook.Description,
				URL:         callbackURL,
				Active:      true,
				Events:      []string{webhookEvent},
				Secret:      s.config.Secret,
			},
		)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update webhook: %w", updateErr)
		}

		return newWebhookStatus(updated, callbackURL), nil
	}

	created, createErr := s.client.CreateWebhook(ctx, workspace, repoSlug, bitbucket.CreateWebhookRequest{
		Description: webhookDescription,
		URL:         callbackURL,
		Active:      true,
		Events:      []string{webhookEvent},
		Secret:      s.config.Secret,
	})
	if createErr != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", createErr)
	}

	return newWebhookStatus(created, callbackURL), nil
}

// UnregisterWebhook removes the Bitbucket webhook pointing at the configured
// callback URL from the project repository. It is a no-op when no such
// webhook exists. Webhook state is never written to the database.
func (s *ManagementService) UnregisterWebhook(ctx context.Context, project *projects.Project) (*WebhookStatus, error) {
	workspace, repoSlug, err := projects.ParseBitbucketRepoURL(project.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	callbackURL := s.webhookCallbackURL()
	hooks, err := s.client.ListWebhooks(ctx, workspace, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	if hook := findWebhookByURL(hooks, callbackURL); hook != nil {
		if deleteErr := s.client.DeleteWebhook(ctx, workspace, repoSlug, hook.UUID); deleteErr != nil {
			return nil, fmt.Errorf("failed to delete webhook: %w", deleteErr)
		}
	}

	return &WebhookStatus{
		Status:      WebhookStatusNotRegistered,
		HookUUID:    "",
		CallbackURL: callbackURL,
		UpdatedAt:   "",
	}, nil
}

// GetWebhookStatus derives the current Bitbucket webhook registration state
// of the project repository. It never reads or writes webhook state in the
// database.
func (s *ManagementService) GetWebhookStatus(ctx context.Context, project *projects.Project) (*WebhookStatus, error) {
	workspace, repoSlug, err := projects.ParseBitbucketRepoURL(project.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	callbackURL := s.webhookCallbackURL()
	hooks, err := s.client.ListWebhooks(ctx, workspace, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	if hook := findWebhookByURL(hooks, callbackURL); hook != nil {
		return newWebhookStatus(hook, callbackURL), nil
	}

	return &WebhookStatus{
		Status:      WebhookStatusNotRegistered,
		HookUUID:    "",
		CallbackURL: callbackURL,
		UpdatedAt:   "",
	}, nil
}

// webhookCallbackURL returns the full callback URL Bitbucket is expected to
// deliver push events to.
func (s *ManagementService) webhookCallbackURL() string {
	return s.callbackBaseURL + webhookCallbackPath
}

// findWebhookByURL returns the first webhook whose URL matches the callback
// URL, or nil.
func findWebhookByURL(hooks []bitbucket.Webhook, callbackURL string) *bitbucket.Webhook {
	for i := range hooks {
		if hooks[i].URL == callbackURL {
			return &hooks[i]
		}
	}

	return nil
}

// newWebhookStatus builds the registered state from a Bitbucket webhook.
func newWebhookStatus(hook *bitbucket.Webhook, callbackURL string) *WebhookStatus {
	return &WebhookStatus{
		Status:      WebhookStatusRegistered,
		HookUUID:    hook.UUID,
		CallbackURL: callbackURL,
		UpdatedAt:   hook.CreatedAt,
	}
}
