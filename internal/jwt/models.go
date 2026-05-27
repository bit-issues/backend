package jwt

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type refreshTokenModel struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    int64     `bun:"user_id"`
	TokenHash string    `bun:"token_hash"`
	ExpiresAt time.Time `bun:"expires_at"`
	Revoked   bool      `bun:"revoked"`
	CreatedAt time.Time `bun:"created_at"`
}

func newRefreshTokenModel(userID int64, tokenHash string, expiresAt time.Time) *refreshTokenModel {
	return &refreshTokenModel{
		BaseModel: schema.BaseModel{},

		ID:        0,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	}
}

func (m *refreshTokenModel) toDomain() *Token {
	if m == nil {
		return nil
	}
	return &Token{
		UserID:    m.UserID,
		Value:     m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		Revoked:   m.Revoked,
	}
}
