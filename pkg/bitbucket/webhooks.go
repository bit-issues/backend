package bitbucket

import "time"

// Hook is a Bitbucket Cloud repository webhook as returned by the API v2
// /repositories/{workspace}/{repo_slug}/hooks endpoints.
type Hook struct {
	UUID        string    `json:"uuid"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Active      bool      `json:"active"`
	Secret      string    `json:"secret,omitempty"`
	Events      []string  `json:"events"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
}

// HookInput is the payload for creating or updating a repository webhook.
// Secret is write-only: Bitbucket never returns it in responses.
type HookInput struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Secret      string   `json:"secret,omitempty"`
	Events      []string `json:"events"`
}

// hookListResponse is the paginated Bitbucket hooks list response.
type hookListResponse struct {
	Values []Hook `json:"values"`
	Next   string `json:"next"`
}
