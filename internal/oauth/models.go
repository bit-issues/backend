package oauth

import (
	"time"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
)

type tokenModel struct {
	bun.BaseModel `bun:"table:oauth_tokens,alias:ot"`
	db.TimedModel

	ID           int64     `bun:"id,pk,autoincrement"`
	UserID       int64     `bun:"user_id"`
	AccessToken  string    `bun:"access_token,notnull"`
	RefreshToken string    `bun:"refresh_token,notnull"`
	Scopes       string    `bun:"scopes,notnull"`
	ExpiresAt    time.Time `bun:"expires_at"`
}

func (t *tokenModel) toDomain() *Token {
	return &Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    t.ExpiresAt,
		Scopes:       t.Scopes,

		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func newTokenModel(userID int64, token *Token) *tokenModel {
	return &tokenModel{
		BaseModel: bun.BaseModel{},
		TimedModel: db.TimedModel{
			CreatedAt: token.CreatedAt,
			UpdatedAt: token.UpdatedAt,
		},

		ID:           0,
		UserID:       userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Scopes:       token.Scopes,
		ExpiresAt:    token.ExpiresAt,
	}
}
