package dto

import (
	"time"

	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
	"github.com/samber/lo"
)

// UserBrief is the shared client-go wire type. The alias keeps existing
// dto.UserBrief references (e.g. history responses) compiling without a duplicate.
type UserBrief = clientgo.UserBrief

type UserBriefList struct {
	Items []UserBrief `json:"items"`
	Total int         `json:"total"`
}

func ToUserBrief(u *users.User) UserBrief {
	return UserBrief{
		ID:        u.ID,
		Name:      u.Name,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

func ResolveUserBrief(id *int64, lookup map[int64]users.User) *UserBrief {
	if id == nil {
		return nil
	}

	if u, ok := lookup[*id]; ok {
		return lo.ToPtr(ToUserBrief(&u))
	}

	return &UserBrief{
		ID:        *id,
		Name:      "",
		Role:      "",
		CreatedAt: "",
	}
}

func ToUserBriefList(items []users.User, total int) UserBriefList {
	return UserBriefList{
		Items: lo.Map(items, func(u users.User, _ int) UserBrief { return ToUserBrief(&u) }),
		Total: total,
	}
}

func ToProject(p *projects.Project) *clientgo.Project {
	if p == nil {
		return nil
	}

	return &clientgo.Project{
		ID:        p.ID,
		Name:      p.Name,
		RepoURL:   p.RepoURL,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
}
