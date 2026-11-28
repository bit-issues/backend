package users

import (
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
	"github.com/samber/lo"
)

const (
	defaultLimit = 20
)

func defaultSearchQuery() clientgo.SearchQuery {
	return clientgo.SearchQuery{
		PaginationQuery: clientgo.PaginationQuery{
			Limit:  defaultLimit,
			Offset: 0,
		},

		Query: "",
	}
}

func defaultListFilter() clientgo.ListFilter {
	return clientgo.ListFilter{
		Status: nil,
		Role:   nil,
		Limit:  defaultLimit,
		Offset: 0,
	}
}

func toUserStatus(value *string) *users.Status {
	if value == nil {
		return nil
	}

	status := users.Status(*value)
	return &status
}

func toUserRole(value *string) *users.Role {
	if value == nil {
		return nil
	}

	role := users.Role(*value)
	return &role
}

// toGetResponse converts domain User to the shared client-go wire DTO.
func toGetResponse(u *users.User) clientgo.UserResponseDTO {
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

func toListResponse(items []users.User, total int) clientgo.ListResponse {
	return clientgo.ListResponse{
		Items: lo.Map(items, func(u users.User, _ int) clientgo.UserResponseDTO { return toGetResponse(&u) }),
		Total: total,
	}
}
