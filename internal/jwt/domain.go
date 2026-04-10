package jwt

import (
	"github.com/bit-issues/backend/internal/users"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT token claims.
type Claims struct {
	jwt.RegisteredClaims

	UserID int64        `json:"user_id"`
	Role   users.Role   `json:"role"`
	Status users.Status `json:"status"`
}
