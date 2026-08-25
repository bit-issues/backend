package oauth

import (
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/oauth"
)

// AuthorizeResponse is the JSON body of the authorize endpoint.
type AuthorizeResponse struct {
	URL string `json:"url" example:"https://bitbucket.org/site/oauth2/authorize?client_id=...&response_type=code&scope=webhook&state=...&redirect_uri=..."`
}

// StatusResponse reports the current OAuth connection state. Optional fields
// are omitted when the connection is absent.
type StatusResponse struct {
	Connected   bool     `json:"connected"`
	ConnectedAt *string  `json:"connected_at,omitempty" example:"2026-08-24T12:00:00Z"`
	ExpiresAt   *string  `json:"expires_at,omitempty"   example:"2026-08-24T14:00:00Z"`
	Scopes      []string `json:"scopes,omitempty"       example:"webhook"`
}

// NewStatusResponse maps a stored OAuth token to the status DTO.
func NewStatusResponse(token *oauth.Token) StatusResponse {
	if token == nil {
		return StatusResponse{
			Connected:   false,
			ConnectedAt: nil,
			ExpiresAt:   nil,
			Scopes:      nil,
		}
	}

	response := StatusResponse{
		Connected:   true,
		ConnectedAt: nil,
		ExpiresAt:   nil,
		Scopes:      nil,
	}
	if !token.CreatedAt.IsZero() {
		connectedAt := token.CreatedAt.Format(time.RFC3339)
		response.ConnectedAt = &connectedAt
	}
	if !token.ExpiresAt.IsZero() {
		expiresAt := token.ExpiresAt.Format(time.RFC3339)
		response.ExpiresAt = &expiresAt
	}
	if scopes := strings.Fields(token.Scope); len(scopes) > 0 {
		response.Scopes = scopes
	}

	return response
}
