package webhooks

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/bit-issues/backend/pkg/bitbucket"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// ManagementHandler serves the admin-only project webhook management
// endpoints. It is registered through the group:"handlers" fx path, after the
// JWT middleware.
type ManagementHandler struct {
	svc    *webhooks.ManagementService
	logger *zap.Logger
}

// NewManagementHandler wires the management handler.
func NewManagementHandler(
	svc *webhooks.ManagementService,
	logger *zap.Logger,
) handler.Handler {
	return &ManagementHandler{
		svc:    svc,
		logger: logger,
	}
}

// Register wires the admin-only webhook management routes.
func (h *ManagementHandler) Register(r fiber.Router) {
	group := r.Group("/projects/:slug/webhook", jwtauth.WithRole(users.RoleAdmin), h.errorsHandler)
	group.Get("/", h.status)
	group.Post("/register", h.register)
	group.Delete("/", h.unregister)
}

//	@Summary		Get project webhook status
//	@Description	Returns the live Bitbucket webhook status for a project. Only administrators can perform this action.
//	@Tags			Webhooks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	WebhookStatusResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		409		{object}	fiberfx.ErrorResponse
//	@Failure		422		{object}	fiberfx.ErrorResponse
//	@Failure		502		{object}	fiberfx.ErrorResponse
//	@Failure		503		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook [get]
//
// status returns the live webhook status (admin only).
func (h *ManagementHandler) status(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	st, err := h.svc.Status(c.Context(), user.ID, c.Params("slug"))
	if err != nil {
		return fmt.Errorf("failed to get webhook status: %w", err)
	}

	return c.JSON(NewWebhookStatusResponse(st))
}

//	@Summary		Register project webhook
//	@Description	Creates or updates the Bitbucket push webhook for a project. Only administrators can perform this action.
//	@Tags			Webhooks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	WebhookStatusResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		409		{object}	fiberfx.ErrorResponse
//	@Failure		422		{object}	fiberfx.ErrorResponse
//	@Failure		501		{object}	fiberfx.ErrorResponse
//	@Failure		502		{object}	fiberfx.ErrorResponse
//	@Failure		503		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook/register [post]
//
// register creates or repairs the project webhook (admin only).
func (h *ManagementHandler) register(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	st, err := h.svc.Register(c.Context(), user.ID, c.Params("slug"))
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}

	return c.JSON(NewWebhookStatusResponse(st))
}

//	@Summary		Unregister project webhook
//	@Description	Deletes the Bitbucket push webhook for a project. Only administrators can perform this action.
//	@Tags			Webhooks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path	string	true	"Project ID"
//	@Success		204
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Failure		409	{object}	fiberfx.ErrorResponse
//	@Failure		422	{object}	fiberfx.ErrorResponse
//	@Failure		501	{object}	fiberfx.ErrorResponse
//	@Failure		502	{object}	fiberfx.ErrorResponse
//	@Failure		503	{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook [delete]
//
// unregister deletes the project webhook (admin only).
func (h *ManagementHandler) unregister(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	if err := h.svc.Unregister(c.Context(), user.ID, c.Params("slug")); err != nil {
		return fmt.Errorf("failed to unregister webhook: %w", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// errorsHandler maps management service errors to HTTP responses. Webhook
// endpoints never return 401: upstream Bitbucket 401 maps to 409 reconnect.
func (h *ManagementHandler) errorsHandler(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	return classifyWebhookError(err)
}

// classifyWebhookError is a pure mapping from service errors to HTTP errors.
func classifyWebhookError(err error) error {
	switch {
	case errors.Is(err, webhooks.ErrOAuthDisabled),
		errors.Is(err, webhooks.ErrCallbackURLNotConfigured),
		errors.Is(err, webhooks.ErrCallbackURLInvalid),
		errors.Is(err, webhooks.ErrSecretNotConfigured):
		return fiber.NewError(fiber.StatusNotImplemented, err.Error())
	case errors.Is(err, webhooks.ErrOAuthNotConnected):
		return fiber.NewError(
			fiber.StatusConflict,
			"Bitbucket OAuth is not connected. Reconnect Bitbucket OAuth to manage webhooks.",
		)
	case errors.Is(err, bitbucket.ErrUnauthorized):
		return fiber.NewError(
			fiber.StatusConflict,
			"Bitbucket OAuth session expired. Reconnect Bitbucket OAuth to manage webhooks.",
		)
	case errors.Is(err, bitbucket.ErrForbidden):
		return fiber.NewError(
			fiber.StatusForbidden,
			"Bitbucket API access denied. The connected account needs admin access to the repository.",
		)
	case errors.Is(err, bitbucket.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, "Bitbucket repository or workspace not found.")
	case errors.Is(err, webhooks.ErrNotBitbucketRepo):
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			"Project repository is not a bitbucket.org repository.",
		)
	case errors.Is(err, projects.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, "Project not found.")
	default:
		var apiErr *bitbucket.APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == http.StatusTooManyRequests {
				return fiber.NewError(fiber.StatusServiceUnavailable, "Bitbucket API rate limit exceeded")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Bitbucket API unavailable")
		}
		return fmt.Errorf("failed to manage webhook: %w", err)
	}
}
