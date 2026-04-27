package users

import (
	"strings"
	"time"

	"github.com/go-core-fx/bunfx"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

// userModel is the storage representation bound to Bun and SQL schema.
type userModel struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	bunfx.TimedModel

	ID           int64  `bun:"id,pk,autoincrement"`
	Email        string `bun:"email,notnull,unique"`
	Name         string `bun:"name,notnull"`
	PasswordHash string `bun:"password_hash,notnull"`
	Role         Role   `bun:"role,notnull,default:'user'"`
	Status       Status `bun:"status,notnull,default:'pending'"`
}

func newUserModel(u UserInput, passwordHash string) *userModel {
	now := time.Now()

	return &userModel{
		BaseModel: schema.BaseModel{},
		TimedModel: bunfx.TimedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},

		ID:           0,
		Email:        u.Email,
		Name:         emailUsername(u.Email),
		PasswordHash: passwordHash,
		Role:         u.Role,
		Status:       StatusPending,
	}
}

func (m *userModel) toDomain() *User {
	if m == nil {
		return nil
	}

	return &User{
		ID:        m.ID,
		Email:     m.Email,
		Name:      m.Name,
		Role:      m.Role,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func emailUsername(email string) string {
	// Best-effort parsing; email is expected to contain '@' due to validation elsewhere.
	if local, _, ok := strings.Cut(email, "@"); ok && local != "" {
		return local
	}
	return email
}
