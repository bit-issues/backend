package dto

import (
	"time"

	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
	"github.com/samber/lo"
)

func ToUserBrief(u *users.User) clientgo.UserBrief {
	return clientgo.UserBrief{
		ID:        u.ID,
		Name:      u.Name,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

func ResolveUserBrief(id *int64, lookup map[int64]users.User) *clientgo.UserBrief {
	if id == nil {
		return nil
	}

	if u, ok := lookup[*id]; ok {
		return lo.ToPtr(ToUserBrief(&u))
	}

	return &clientgo.UserBrief{
		ID:        *id,
		Name:      "",
		Role:      "",
		CreatedAt: "",
	}
}

func ToUserBriefList(items []users.User, total int) clientgo.UserBriefList {
	return clientgo.UserBriefList{
		Items: lo.Map(items, func(u users.User, _ int) clientgo.UserBrief { return ToUserBrief(&u) }),
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
