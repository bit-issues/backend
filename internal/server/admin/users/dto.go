package users

import (
	"time"

	"github.com/bit-issues/backend/internal/users"
)

const (
	defaultLimit = 20
)

// AdminUserFilter represents query parameters for filtering users.
//
//	@Description	Query parameters for filtering and paginating users list.
type AdminUserFilter struct {
	Status *users.Status `query:"status" validate:"omitempty,oneof=pending active blocked" enums:"pending,active,blocked"`
	Role   *users.Role   `query:"role"   validate:"omitempty,oneof=admin user"             enums:"admin,user"`
	Limit  int           `query:"limit"  validate:"min=1,max=100"                                                         default:"20"`
	Offset int           `query:"offset" validate:"min=0"                                                                 default:"0"`
}

// UpdateUserRequest represents admin request to update user status/role.
//
//	@Description	Admin can update user status (active/blocked/pending) and role (admin/user).
type UpdateUserRequest struct {
	Status *users.Status `json:"status,omitempty" validate:"omitempty,oneof=pending active blocked" enums:"pending,active,blocked"`
	Role   *users.Role   `json:"role,omitempty"   validate:"omitempty,oneof=admin user"             enums:"admin,user"`
}

// UserResponse represents user data in admin responses (without password).
//
//	@Description	User data returned in admin API responses (password excluded).
type UserResponse struct {
	ID        int64        `json:"id"         example:"42"`
	Email     string       `json:"email"      example:"user@example.com"`
	Role      users.Role   `json:"role"       example:"user"                 enums:"admin,user"`
	Status    users.Status `json:"status"     example:"active"               enums:"pending,active,blocked"`
	CreatedAt time.Time    `json:"created_at" example:"2026-04-06T10:00:00Z"`
	UpdatedAt time.Time    `json:"updated_at" example:"2026-04-06T10:00:00Z"`
}

// UserListResponse represents paginated list of users.
//
//	@Description	Paginated response containing users list and total count.
type UserListResponse struct {
	Items []UserResponse `json:"items"`
	Total int            `json:"total" example:"42"`
}

func defaultAdminUserFilter() AdminUserFilter {
	return AdminUserFilter{
		Status: nil,
		Role:   nil,
		Limit:  defaultLimit,
		Offset: 0,
	}
}

// toUserResponse converts domain User to admin UserResponse.
func toUserResponse(u *users.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
