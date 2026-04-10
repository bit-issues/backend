package auth

import (
	"errors"
	"fmt"

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
	jwtSvc   *jwt.Service
}

func NewHandler(service *users.Service, validate *validator.Validate, jwtSvc *jwt.Service) handler.Handler {
	return &Handler{
		Base:     handler.Base{Validator: validate},
		usersSvc: service,
		jwtSvc:   jwtSvc,
	}
}

func (h *Handler) Register(r fiber.Router) {
	auth := r.Group("/auth", h.errorsHandler)

	auth.Post("/register", validation.DecorateWithBodyEx(h.Validator, h.handleRegister))
	auth.Post("/login", validation.DecorateWithBodyEx(h.Validator, h.handleLogin))
	auth.Post(
		"/change-password",
		jwtauth.New(h.jwtSvc, h.usersSvc),
		validation.DecorateWithBodyEx(h.Validator, h.handleChangePassword),
	)
}

// handleRegister handles user registration.
//
//	@Summary		Register a new user
//	@Description	Register a new user with email and password. The user will be created with status "pending" and requires admin activation.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest			true	"Registration request"
//	@Success		201		{object}	UserResponseDTO			"User created successfully"
//	@Failure		400		{object}	fiberfx.ErrorResponse	"Validation error"
//	@Failure		409		{object}	fiberfx.ErrorResponse	"Email already exists"
//	@Router			/auth/register [post]
func (h *Handler) handleRegister(c *fiber.Ctx, req *RegisterRequest) error {
	user, err := h.usersSvc.Register(c.Context(), users.UserInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     users.RoleUser,
	})
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(toUserResponseDTO(user))
}

// handleLogin handles user login.
//
//	@Summary		User login
//	@Description	Authenticate user with email and password, returns JWT token and user info.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest			true	"Login credentials"
//	@Success		200		{object}	LoginResponse			"Login successful"
//	@Failure		401		{object}	fiberfx.ErrorResponse	"Invalid credentials"
//	@Failure		403		{object}	fiberfx.ErrorResponse	"Account not active"
//	@Router			/auth/login [post]
func (h *Handler) handleLogin(c *fiber.Ctx, req *LoginRequest) error {
	user, err := h.usersSvc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return fmt.Errorf("failed to login user: %w", err)
	}

	token, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	return c.JSON(LoginResponse{
		AccessToken: token,
		User:        toUserResponseDTO(user),
	})
}

// handleChangePassword handles password change for authenticated user.
//
//	@Summary		Change password
//	@Description	Change the authenticated user's password. Requires JWT authentication.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body	ChangePasswordRequest	true	"New password data"
//	@Success		204		"Password changed successfully"
//	@Failure		400		{object}	fiberfx.ErrorResponse	"Validation error"
//	@Failure		401		{object}	fiberfx.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	fiberfx.ErrorResponse	"Forbidden"
//	@Router			/auth/change-password [post]
func (h *Handler) handleChangePassword(c *fiber.Ctx, req *ChangePasswordRequest) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.usersSvc.ChangePassword(c.Context(), user.ID, req.OldPassword, req.NewPassword); err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
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
