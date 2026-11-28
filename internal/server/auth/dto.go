package auth

import (
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

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
