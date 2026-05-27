package jwt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bit-issues/backend/internal/users"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const refreshTokenBytes = 32

// Service handles JWT token operations.
type Service struct {
	config      Config
	refreshRepo *Repository
}

// NewService creates and initializes a new JWT service.
func NewService(config Config, refreshRepo *Repository) *Service {
	return &Service{
		config:      config,
		refreshRepo: refreshRepo,
	}
}

// GenerateTokenPair creates a new access token and refresh token pair.
func (s *Service) GenerateTokenPair(ctx context.Context, user *users.User) (string, string, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) generateAccessToken(user *users.User) (string, error) {
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

// GenerateRefreshToken creates an opaque random refresh token, stores its hash in the database,
// and returns the raw token string.
func (s *Service) GenerateRefreshToken(ctx context.Context, userID int64) (string, error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	rawToken := hex.EncodeToString(raw)
	hash := sha256Hex(rawToken)

	if err := s.refreshRepo.Create(ctx, userID, hash, time.Now().Add(s.config.RefreshTTL)); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return rawToken, nil
}

// ValidateRefreshToken validates a raw refresh token. Returns the user ID if valid.
func (s *Service) ValidateRefreshToken(ctx context.Context, rawToken string) (int64, error) {
	hash := sha256Hex(rawToken)

	model, err := s.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		return 0, err
	}
	if model == nil {
		return 0, ErrInvalidToken
	}
	if model.Revoked {
		return 0, ErrRefreshTokenRevoked
	}
	if time.Now().After(model.ExpiresAt) {
		return 0, ErrExpiredToken
	}

	return model.UserID, nil
}

// RevokeRefreshToken revokes a raw refresh token.
func (s *Service) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	hash := sha256Hex(rawToken)
	return s.refreshRepo.RevokeByHash(ctx, hash)
}

// RotateRefreshToken revokes the old token and generates a new one.
// Returns the new raw refresh token.
func (s *Service) RotateRefreshToken(ctx context.Context, oldRawToken string, userID int64) (string, error) {
	if err := s.RevokeRefreshToken(ctx, oldRawToken); err != nil {
		return "", fmt.Errorf("failed to revoke old refresh token: %w", err)
	}
	return s.GenerateRefreshToken(ctx, userID)
}

// RotateTokenPair revokes the old refresh token and generates a new access token + refresh token pair.
func (s *Service) RotateTokenPair(ctx context.Context, oldRawToken string, user *users.User) (string, string, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.RotateRefreshToken(ctx, oldRawToken, user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func sha256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}
