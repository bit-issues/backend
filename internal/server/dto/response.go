package dto

import (
	"time"

	"github.com/bit-issues/backend/internal/projects"
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

// Project represents the API response for a single project.
type Project struct {
	ID        string `json:"id"         example:"backend-service"`
	Name      string `json:"name"       example:"Backend Service"`
	RepoURL   string `json:"repo_url"   example:"https://bitbucket.org/company/backend"`
	CreatedAt string `json:"created_at" example:"2026-04-01T08:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-04-02T09:00:00Z"`
}

func ToProject(p *projects.Project) *Project {
	if p == nil {
		return nil
	}

	return &Project{
		ID:        p.ID,
		Name:      p.Name,
		RepoURL:   p.RepoURL,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
}
