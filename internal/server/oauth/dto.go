package oauth

import (
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/oauth"
	clientgo "github.com/bit-issues/client-go"
)

// NewStatusResponse maps a stored OAuth token to the status DTO.
func NewStatusResponse(token *oauth.Token) clientgo.StatusResponse {
	if token == nil {
		return clientgo.StatusResponse{
			Connected:   false,
			ConnectedAt: nil,
			ExpiresAt:   nil,
			Scopes:      nil,
		}
	}

	response := clientgo.StatusResponse{
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
	if scopes := strings.Fields(token.Scopes); len(scopes) > 0 {
		response.Scopes = scopes
	}

	return response
}
