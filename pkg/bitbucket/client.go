package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DefaultBaseURL is the Bitbucket Cloud API v2 base URL.
const DefaultBaseURL = "https://api.bitbucket.org/2.0"

// maxListPages bounds ListHooks pagination to at most this many page requests.
const maxListPages = 10

// Client is a minimal Bitbucket Cloud API v2 client for repository webhooks.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a Client for the given API base URL and HTTP client. An
// empty baseURL selects DefaultBaseURL; a trailing slash is normalized away.
// The HTTP client carries the request timeout, so it must not be a client
// without one.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    httpClient,
	}
}

// ListHooks returns all repository webhooks, following pagination up to
// maxListPages pages.
func (c *Client) ListHooks(ctx context.Context, workspace, repoSlug, token string) ([]Hook, error) {
	base := c.hooksPath(workspace, repoSlug) + "?pagelen=100"

	var hooks []Hook
	for page := 1; page <= maxListPages; page++ {
		path := base
		if page > 1 {
			path += "&page=" + strconv.Itoa(page)
		}

		var resp hookListResponse
		if err := c.do(ctx, http.MethodGet, path, token, nil, &resp); err != nil {
			return nil, err
		}

		hooks = append(hooks, resp.Values...)
		if resp.Next == "" {
			break
		}
	}

	return hooks, nil
}

// CreateHook registers a new repository webhook.
func (c *Client) CreateHook(ctx context.Context, workspace, repoSlug, token string, input HookInput) (*Hook, error) {
	var hook Hook
	if err := c.do(ctx, http.MethodPost, c.hooksPath(workspace, repoSlug), token, input, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// UpdateHook updates the repository webhook with the given uuid. The uuid is
// URL-escaped, including its braces.
func (c *Client) UpdateHook(
	ctx context.Context,
	workspace, repoSlug, hookUUID, token string,
	input HookInput,
) (*Hook, error) {
	var hook Hook
	path := c.hooksPath(workspace, repoSlug) + "/" + url.PathEscape(hookUUID)
	if err := c.do(ctx, http.MethodPut, path, token, input, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// DeleteHook removes the repository webhook with the given uuid. The uuid is
// URL-escaped, including its braces.
func (c *Client) DeleteHook(ctx context.Context, workspace, repoSlug, hookUUID, token string) error {
	path := c.hooksPath(workspace, repoSlug) + "/" + url.PathEscape(hookUUID)
	return c.do(ctx, http.MethodDelete, path, token, nil, nil)
}

// hooksPath builds the hooks collection path for a repository.
func (c *Client) hooksPath(workspace, repoSlug string) string {
	return "/repositories/" + url.PathEscape(workspace) + "/" + url.PathEscape(repoSlug) + "/hooks"
}

// do performs an authenticated JSON request against the API. Response bodies
// of failed requests are never read, so upstream content cannot leak into
// error text.
func (c *Client) do(ctx context.Context, method, path, token string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("failed to build bitbucket request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return &APIError{StatusCode: 0, Message: "bitbucket api unavailable"}
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusForbidden:
		return ErrForbidden
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
		if out == nil || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		err = json.NewDecoder(resp.Body).Decode(out)
		if err != nil {
			return fmt.Errorf("failed to decode bitbucket response: %w", err)
		}
		return nil
	default:
		return &APIError{StatusCode: resp.StatusCode, Message: "bitbucket api error"}
	}
}
