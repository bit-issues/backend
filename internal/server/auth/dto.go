package auth

import (
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

// RegisterRequest represents user registration data.
//
//	@Description	User registration request with email and password.
type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// ChangePasswordRequest represents password change request.
//
//	@Description	Request to change user's password (requires old password for verification).
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

// ToUserResponseDTO converts domain User to the shared wire DTO.
func ToUserResponseDTO(u *users.User) clientgo.UserResponseDTO {
	return clientgo.UserResponseDTO{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		Status:    string(u.Status),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
