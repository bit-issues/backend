package users

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
)

type UserInput struct {
	Email    string
	Password string
	Role     Role
}

type UserUpdate struct {
	Status *Status
	Role   *Role
}

// User is the domain entity used by service and handler layers.
type User struct {
	ID        int64
	Email     string
	Role      Role
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserWithPasswordHash struct {
	User

	PasswordHash string
}

func (u UserUpdate) IsEmpty() bool {
	return u.Status == nil && u.Role == nil
}

// IsValidRole checks if the role is one of the allowed values.
func IsValidRole(r Role) bool {
	switch r {
	case RoleAdmin, RoleUser:
		return true
	}
	return false
}

// IsValidStatus checks if the status is one of the allowed values.
func IsValidStatus(s Status) bool {
	switch s {
	case StatusPending, StatusActive, StatusBlocked:
		return true
	}
	return false
}
