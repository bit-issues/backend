package auth

import (
	"time"

	"github.com/bit-issues/backend/internal/users"
)

// RegisterRequest represents user registration data.
//
//	@Description	User registration request with email and password.
type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// LoginRequest represents user login data.
//
//	@Description	User login credentials.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents successful login response.
//
//	@Description	Successful login response containing JWT token, refresh token, and user info.
type LoginResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	User         UserResponseDTO `json:"user"`
}

// RefreshRequest represents a token refresh request.
//
//	@Description	Request to refresh an access token using a refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshResponse represents a successful token refresh response.
//
//	@Description	Response containing a new access token and a rotated refresh token.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest represents a logout request.
//
//	@Description	Request to invalidate a refresh token on logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UserResponseDTO represents user data in responses (without password).
//
//	@Description	User data returned in API responses (password excluded).
type UserResponseDTO struct {
	ID        int64        `json:"id"`
	Email     string       `json:"email"`
	Name      string       `json:"name"`
	Role      users.Role   `json:"role"`
	Status    users.Status `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ChangePasswordRequest represents password change request.
//
//	@Description	Request to change user's password (requires old password for verification).
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

// ToUserResponseDTO converts domain User to response DTO.
func ToUserResponseDTO(u *users.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
