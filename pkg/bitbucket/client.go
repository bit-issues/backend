package bitbucket

import (
	"context"
	"fmt"
	"net/http"

	restkit "github.com/capcom6/go-restkit"
)

// defaultBaseURL is the public Bitbucket REST API root.
const defaultBaseURL = "https://api.bitbucket.org"

// Client talks to the Bitbucket REST API using Bearer token authentication.
type Client struct {
	rest        *restkit.Client
	accessToken string
}

// NewClient creates a Bitbucket API client. When cfg.BaseURL is empty the
// public Bitbucket API is used.
func NewClient(cfg Config) (*Client, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	restClient, err := restkit.NewClient(restkit.Config{
		Client:  http.DefaultClient,
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &Client{
		rest:        restClient,
		accessToken: cfg.AccessToken,
	}, nil
}

// do performs an authenticated request against the Bitbucket API. Errors are
// wrapped to add context; the go-restkit error types remain reachable through
// the chain via [errors.As] helpers (IsClientError, IsServerError,
// IsInfrastructureError, AsAPIError).
func (c *Client) do(ctx context.Context, method, path string, payload, response any) error {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.accessToken)

	if err := c.rest.Do(ctx, method, path, headers, payload, response); err != nil {
		return fmt.Errorf("bitbucket API request failed: %w", err)
	}

	return nil
}
