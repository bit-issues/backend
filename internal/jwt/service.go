package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bit-issues/backend/internal/users"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Service handles JWT token operations.
type Service struct {
	config Config
}

// NewService creates and initializes a new JWT service.
func NewService(config Config) *Service {
	return &Service{
		config: config,
	}
}

// GenerateToken creates a new JWT token for the user.
func (s *Service) GenerateToken(user *users.User) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID: user.ID,
		Role:   user.Role,
		Status: user.Status,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates and parses a JWT token.
// Returns the claims if valid, or an error if invalid/expired.
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		new(Claims),
		func(_ *jwt.Token) (any, error) {
			return []byte(s.config.Secret), nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(s.config.Issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
