package passkey

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/server/auth"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/internal/webauthn"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	handler.Base

	waSvc  *webauthn.Service
	jwtSvc *jwt.Service
}

func NewHandler(
	waSvc *webauthn.Service,
	jwtSvc *jwt.Service,
	validate *validator.Validate,
) handler.Handler {
	return &Handler{
		Base:   handler.Base{Validator: validate},
		waSvc:  waSvc,
		jwtSvc: jwtSvc,
	}
}

func (h *Handler) Register(r fiber.Router) {
	passkey := r.Group("/auth/passkey", h.errorsHandler)

	passkey.Post("/register/begin", h.handlePasskeyRegisterBegin)
	passkey.Post("/register/complete", h.handlePasskeyRegisterComplete)
	passkey.Post("/login/begin", h.handlePasskeyLoginBegin)
	passkey.Post("/login/complete", h.handlePasskeyLoginComplete)
	passkey.Get("/credentials", h.handleListPasskeyCredentials)
	passkey.Patch("/credentials/:id", h.handleRenamePasskeyCredential)
	passkey.Delete("/credentials/:id", h.handleDeletePasskeyCredential)
}

func (h *Handler) handlePasskeyRegisterBegin(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	options, err := h.waSvc.BeginRegistration(c.Context(), user)
	if err != nil {
		return fmt.Errorf("failed to begin passkey registration: %w", err)
	}

	return c.JSON(options)
}

func (h *Handler) handlePasskeyRegisterComplete(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	body := c.Body()
	if len(body) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty request body")
	}

	cred, err := h.waSvc.FinishRegistration(c.Context(), user, body)
	if err != nil {
		return fmt.Errorf("failed to complete passkey registration: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(CredentialResponse{
		ID:        cred.ID,
		Name:      cred.Name,
		CreatedAt: cred.CreatedAt,
	})
}

func (h *Handler) handlePasskeyLoginBegin(c *fiber.Ctx) error {
	options, err := h.waSvc.BeginLogin(c.Context())
	if err != nil {
		return fmt.Errorf("failed to begin passkey login: %w", err)
	}

	return c.JSON(options)
}

func (h *Handler) handlePasskeyLoginComplete(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty request body")
	}

	user, err := h.waSvc.FinishLogin(c.Context(), body)
	if err != nil {
		return fmt.Errorf("failed to complete passkey login: %w", err)
	}

	if user.Status != users.StatusActive {
		return fiber.NewError(fiber.StatusForbidden, "user is not active")
	}

	accessToken, refreshToken, err := h.jwtSvc.GenerateTokenPair(c.Context(), user)
	if err != nil {
		return fmt.Errorf("failed to generate token pair: %w", err)
	}

	return c.JSON(auth.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         auth.ToUserResponseDTO(user),
	})
}

func (h *Handler) handleListPasskeyCredentials(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	creds, err := h.waSvc.GetCredentials(c.Context(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to list passkey credentials: %w", err)
	}

	resp := make([]CredentialResponse, 0, len(creds))
	for _, cred := range creds {
		resp = append(resp, CredentialResponse{
			ID:        cred.ID,
			Name:      cred.Name,
			CreatedAt: cred.CreatedAt,
		})
	}

	return c.JSON(resp)
}

func (h *Handler) handleRenamePasskeyCredential(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid credential id")
	}

	var req RenameRequest
	if bodyErr := h.BodyParserValidator(c, &req); bodyErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if rnErr := h.waSvc.RenameCredential(c.Context(), id, user.ID, req.Name); rnErr != nil {
		return fmt.Errorf("failed to rename credential: %w", rnErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) handleDeletePasskeyCredential(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid credential id")
	}

	if delErr := h.waSvc.DeleteCredential(c.Context(), id, user.ID); delErr != nil {
		return fmt.Errorf("failed to delete credential: %w", delErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) errorsHandler(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, webauthn.ErrSessionNotFound):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, webauthn.ErrCredentialNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, webauthn.ErrInvalidWebAuthnPayload):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, jwt.ErrInvalidConfig):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())

	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}
