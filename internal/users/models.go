package users

import (
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
		PasswordHash: passwordHash,
		Role:         u.Role,
		Status:       StatusPending,
	}
}

func (m *userModel) toDomain() *UserWithPasswordHash {
	if m == nil {
		return nil
	}

	return &UserWithPasswordHash{
		User: User{
			ID:        m.ID,
			Email:     m.Email,
			Role:      m.Role,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		PasswordHash: m.PasswordHash,
	}
}
