package dto

// UserBrief represents a minimal user profile for task author/assignee.
//
//	@Description	Minimal user information for task relationships.
type UserBrief struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}
