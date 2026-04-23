package dto

import (
	"time"

	"github.com/bit-issues/backend/internal/users"
	"github.com/samber/lo"
)

// UserBrief represents a minimal user profile for task author/assignee.
//
//	@Description	Minimal user information for task relationships.
type UserBrief struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func ToUserBrief(u *users.User) UserBrief {
	return UserBrief{
		ID:        u.ID,
		Name:      u.Name,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

type UserBriefList struct {
	Items []UserBrief `json:"items"`
	Total int         `json:"total"`
}

func ToUserBriefList(items []users.User, total int) UserBriefList {
	return UserBriefList{
		Items: lo.Map(items, func(u users.User, _ int) UserBrief { return ToUserBrief(&u) }),
		Total: total,
	}
}
