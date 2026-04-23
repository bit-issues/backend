package users

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	handler.Base

	usersSvc *users.Service

	jwtSvc *jwt.Service
}

func NewHandler(usersSvc *users.Service, jwtSvc *jwt.Service, validate *validator.Validate) handler.Handler {
	return &Handler{
		Base: handler.Base{Validator: validate},

		usersSvc: usersSvc,

		jwtSvc: jwtSvc,
	}
}

func (h *Handler) Register(r fiber.Router) {
	group := r.Group(
		"/users",
		h.errorsHandler,
	)

	// GET /users - list all users with optional filters
	group.Get("/", jwtauth.WithRole(users.RoleAdmin), h.handleList)

	// GET /users/search - search active users by name
	group.Get("/search", h.handleSearch)

	// PATCH /users/{id} - update user status/role
	group.Patch("/:id", jwtauth.WithRole(users.RoleAdmin), validation.DecorateWithBodyEx(h.Validator, h.handleUpdate))
}

// handleList returns a paginated list of users with optional status filter.
//
//	@Summary		List all users
//	@Description	List all users with optional status filter.
//	@Tags			Admin, Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		users.Status			false	"Filter by status"
//	@Param			role	query		users.Role				false	"Filter by role"
//	@Param			limit	query		int						false	"Page limit"	default(20)
//	@Param			offset	query		int						false	"Page offset"	default(0)
//	@Success		200		{object}	ListResponse			"Users list"
//	@Failure		401		{object}	fiberfx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	fiberfx.ErrorResponse	"Forbidden"
//	@Router			/users [get]
func (h *Handler) handleList(c *fiber.Ctx) error {
	filter := defaultListFilter()

	if err := h.QueryParserValidator(c, &filter); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	// Get users from service
	usersList, total, err := h.usersSvc.List(
		c.Context(),
		filter.Status,
		filter.Role,
		users.NewPagination(filter.Limit, filter.Offset),
	)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	return c.JSON(toListResponse(usersList, total))
}

// handleSearch returns a paginated list of active users filtered by user name prefix.
//
//	@Summary		Search active users by name
//	@Description	Search active users by name. Accessible for all authorized users.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			query	query		string					true	"Search query"
//	@Param			limit	query		int						false	"Page limit"	default(20)
//	@Param			offset	query		int						false	"Page offset"	default(0)
//	@Success		200		{object}	dto.UserBriefList		"Active users list"
//	@Failure		401		{object}	fiberfx.ErrorResponse	"Unauthorized"
//	@Router			/users/search [get]
func (h *Handler) handleSearch(c *fiber.Ctx) error {
	query := defaultSearchQuery()
	if err := h.QueryParserValidator(c, &query); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	usersList, total, err := h.usersSvc.Search(
		c.Context(),
		query.Query,
		users.NewPagination(query.Limit, query.Offset),
	)
	if err != nil {
		return fmt.Errorf("failed to search users: %w", err)
	}

	return c.JSON(dto.ToUserBriefList(usersList, total))
}

// handleUpdate updates user status and/or role by admin.
//
//	@Summary		Update user
//	@Description	Admin can update user status (active/blocked/pending) and role (admin/user).
//	@Tags			Admin, Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64					true	"User ID"
//	@Param			request	body		UpdateRequest			true	"Update data"
//	@Success		200		{object}	GetResponse				"Updated user"
//	@Failure		400		{object}	fiberfx.ErrorResponse	"Validation error"
//	@Failure		401		{object}	fiberfx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	fiberfx.ErrorResponse	"Forbidden"
//	@Failure		404		{object}	fiberfx.ErrorResponse	"User not found"
//	@Router			/users/{id} [patch]
func (h *Handler) handleUpdate(c *fiber.Ctx, req *UpdateRequest) error {
	// Parse user ID from path
	idStr := c.Params("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	// Validate at least one field is provided
	if req.Status == nil && req.Role == nil {
		return fiber.NewError(fiber.StatusBadRequest, "at least one of status or role must be provided")
	}

	// Perform update
	if updErr := h.usersSvc.Update(
		c.Context(),
		userID,
		users.UserUpdate{Status: req.Status, Role: req.Role},
	); updErr != nil {
		if errors.Is(updErr, users.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, updErr.Error())
		}
		return fmt.Errorf("failed to update user: %w", updErr)
	}

	// Fetch updated user
	updatedUser, err := h.usersSvc.GetByID(c.Context(), userID)
	if err != nil {
		return fmt.Errorf("failed to fetch updated user: %w", err)
	}

	return c.JSON(toGetResponse(updatedUser))
}

// errorsHandler converts service errors to HTTP errors.
func (h *Handler) errorsHandler(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, users.ErrEmailAlreadyUsed):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, users.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, users.ErrInvalidCredential):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, users.ErrNotActive):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, users.ErrEmptyQuery):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())

	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}
