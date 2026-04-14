package users

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/jwt"
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
	admin := r.Group(
		"/users",
		h.errorsHandler,
		jwtauth.WithRole(users.RoleAdmin),
	)

	// GET /users - list all users with optional filters
	admin.Get("/", h.handleList)

	// PATCH /users/{id} - update user status/role
	admin.Patch("/:id", validation.DecorateWithBodyEx(h.Validator, h.handleUpdate))
}

// handleList returns a paginated list of users with optional status filter.
//
//	@Summary		List all users
//	@Description	Admin can list all users with optional status filter and pagination.
//	@Tags			Admin
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
	usersList, err := h.usersSvc.List(c.Context(), filter.Status, filter.Role, filter.Limit, filter.Offset)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Get total count for pagination
	total, err := h.usersSvc.Count(c.Context(), filter.Status, filter.Role)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	// Convert to response DTOs
	items := make([]GetResponse, 0, len(usersList))
	for _, u := range usersList {
		items = append(items, toGetResponse(&u))
	}

	return c.JSON(ListResponse{
		Items: items,
		Total: int(total),
	})
}

// handleUpdate updates user status and/or role by admin.
//
//	@Summary		Update user
//	@Description	Admin can update user status (active/blocked/pending) and role (admin/user).
//	@Tags			Admin
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

	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}
