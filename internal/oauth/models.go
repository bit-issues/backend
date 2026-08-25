package oauth

import (
	"time"

	"github.com/go-core-fx/bunfx"
	"github.com/uptrace/bun"
)

// SingletonID is the fixed primary key of the single-tenant OAuth token row.
// The system holds exactly one active Bitbucket OAuth credential.
const SingletonID uint8 = 1

// Token is the domain representation of the stored Bitbucket OAuth
// credential. Access and refresh tokens are stored plaintext for the MVP.
type Token struct {
	AccessToken       string
	RefreshToken      string
	Scope             string
	ExpiresAt         time.Time
	ConnectedByUserID int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// State is the domain representation of a CSRF state bound to an admin user
// and the OAuth redirect URI. Only the SHA-256 hash is persisted.
type State struct {
	StateHash   string
	UserID      int64
	RedirectURI string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

type tokenModel struct {
	bun.BaseModel `bun:"table:oauth_tokens,alias:ot"`
	bunfx.TimedModel

	SingletonID       uint8     `bun:"singleton_id,pk"`
	AccessToken       string    `bun:"access_token,notnull"`
	RefreshToken      string    `bun:"refresh_token,notnull"`
	Scope             string    `bun:"scope,notnull"`
	ExpiresAt         time.Time `bun:"expires_at,notnull"`
	ConnectedByUserID int64     `bun:"connected_by_user_id,notnull"`
}

func newTokenModel(token *Token) *tokenModel {
	now := time.Now()

	return &tokenModel{
		BaseModel: bun.BaseModel{},
		TimedModel: bunfx.TimedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},

		SingletonID:       SingletonID,
		AccessToken:       token.AccessToken,
		RefreshToken:      token.RefreshToken,
		Scope:             token.Scope,
		ExpiresAt:         token.ExpiresAt,
		ConnectedByUserID: token.ConnectedByUserID,
	}
}

func (m *tokenModel) toDomain() *Token {
	if m == nil {
		return nil
	}

	return &Token{
		AccessToken:       m.AccessToken,
		RefreshToken:      m.RefreshToken,
		Scope:             m.Scope,
		ExpiresAt:         m.ExpiresAt,
		ConnectedByUserID: m.ConnectedByUserID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

type stateModel struct {
	bun.BaseModel `bun:"table:oauth_states,alias:os"`

	StateHash   string    `bun:"state_hash,pk"`
	UserID      int64     `bun:"user_id,notnull"`
	RedirectURI string    `bun:"redirect_uri,notnull"`
	ExpiresAt   time.Time `bun:"expires_at,notnull"`
	CreatedAt   time.Time `bun:"created_at,notnull"`
}

func newStateModel(state *State) *stateModel {
	return &stateModel{
		BaseModel:   bun.BaseModel{},
		StateHash:   state.StateHash,
		UserID:      state.UserID,
		RedirectURI: state.RedirectURI,
		ExpiresAt:   state.ExpiresAt,
		CreatedAt:   state.CreatedAt,
	}
}

func (m *stateModel) toDomain() *State {
	if m == nil {
		return nil
	}

	return &State{
		StateHash:   m.StateHash,
		UserID:      m.UserID,
		RedirectURI: m.RedirectURI,
		ExpiresAt:   m.ExpiresAt,
		CreatedAt:   m.CreatedAt,
	}
}
