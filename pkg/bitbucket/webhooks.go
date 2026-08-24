package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Webhook represents a Bitbucket repository webhook. The secret value is never
// returned by Bitbucket; only SecretSet indicates whether one is configured.
type Webhook struct {
	UUID        string   `json:"uuid"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
	CreatedAt   string   `json:"created_at"`
	SecretSet   bool     `json:"secret_set"`
}

// CreateWebhookRequest is the payload for creating a repository webhook.
type CreateWebhookRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
	Secret      string   `json:"secret"`
}

// UpdateWebhookRequest is the payload for updating an existing webhook.
type UpdateWebhookRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
	Secret      string   `json:"secret"`
}

// webhooksPage is one page of the paginated webhooks listing.
type webhooksPage struct {
	Values []Webhook `json:"values"`
	Next   string    `json:"next"`
}

func hooksPath(workspace, repoSlug string) string {
	return fmt.Sprintf(
		"/2.0/repositories/%s/%s/hooks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
}

func webhookPath(workspace, repoSlug, uuid string) string {
	return fmt.Sprintf("%s/%s", hooksPath(workspace, repoSlug), url.PathEscape(uuid))
}

// CreateWebhook registers a new webhook on the given repository.
func (c *Client) CreateWebhook(
	ctx context.Context,
	workspace, repoSlug string,
	req CreateWebhookRequest,
) (*Webhook, error) {
	var webhook Webhook
	if err := c.do(ctx, http.MethodPost, hooksPath(workspace, repoSlug), req, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// ListWebhooks returns all webhooks of the given repository, following the
// opaque pagination links returned by Bitbucket until exhausted.
func (c *Client) ListWebhooks(ctx context.Context, workspace, repoSlug string) ([]Webhook, error) {
	var result []Webhook

	path := hooksPath(workspace, repoSlug)
	for path != "" {
		var page webhooksPage
		if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}

		result = append(result, page.Values...)
		path = page.Next
	}

	return result, nil
}

// UpdateWebhook replaces the configuration of an existing webhook.
func (c *Client) UpdateWebhook(
	ctx context.Context,
	workspace, repoSlug, uuid string,
	req UpdateWebhookRequest,
) (*Webhook, error) {
	var webhook Webhook
	if err := c.do(ctx, http.MethodPut, webhookPath(workspace, repoSlug, uuid), req, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// DeleteWebhook removes a webhook from the given repository.
func (c *Client) DeleteWebhook(ctx context.Context, workspace, repoSlug, uuid string) error {
	return c.do(ctx, http.MethodDelete, webhookPath(workspace, repoSlug, uuid), nil, nil)
}
