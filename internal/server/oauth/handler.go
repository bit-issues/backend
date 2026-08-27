package oauth

import (
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler serves the Bitbucket OAuth connection lifecycle endpoints.
type Handler struct {
	oauthSvc *oauth.Service
	logger   *zap.Logger
}

// NewHandler creates the OAuth HTTP handler.
func NewHandler(
	oauthSvc *oauth.Service,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		oauthSvc: oauthSvc,
		logger:   logger,
	}
}

func (h *Handler) RegisterPublic(r fiber.Router) {
	r.Get("/oauth/bitbucket/callback", h.callback)
}

func (h *Handler) Register(r fiber.Router) {
	group := r.Group("/oauth/bitbucket", jwtauth.WithRole(users.RoleAdmin))
	group.Get("/authorize", h.authorize)
	group.Get("/status", h.status)
	group.Post("/disconnect", h.disconnect)
}

//	@Summary		Bitbucket OAuth callback
//	@Description	Public callback that exchanges the authorization code with Bitbucket, stores the tokens, and redirects to the admin settings page.
//	@Tags			OAuth
//	@Param			code	query	string	false	"Authorization code"
//	@Param			state	query	string	false	"State"
//	@Param			error	query	string	false	"Bitbucket OAuth error code"
//	@Success		302
//	@Router			/oauth/bitbucket/callback [get]
//
// callback completes the Bitbucket OAuth connection flow (public). All
// outcomes are browser redirects; tokens are never included in them.
func (h *Handler) callback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")

	// Bitbucket declined the authorization (e.g. the admin denied consent).
	if denied := c.Query("error"); denied != "" {
		return h.redirect(c, "?oauth=error&reason=access_denied")
	}

	if state == "" || code == "" {
		return h.redirect(c, "?oauth=error&reason=missing_params")
	}

	if exchErr := h.oauthSvc.Exchange(c.Context(), state, code); exchErr != nil {
		h.logger.Error("oauth callback token exchange failed", zap.Error(exchErr))
		return h.redirect(c, "?oauth=error&reason=exchange_failed")
	}

	return h.redirect(c, "?oauth=success")
}

//	@Summary		Start Bitbucket OAuth connection
//	@Description	Generates a CSRF state and returns the Bitbucket authorization URL. Only administrators can perform this action.
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	AuthorizeResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Router			/oauth/bitbucket/authorize [get]
//
// authorize starts the Bitbucket OAuth connection flow (admin only).
func (h *Handler) authorize(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	return c.JSON(AuthorizeResponse{URL: h.oauthSvc.AuthorizeURL(c.Context(), user.ID)})
}

//	@Summary		Bitbucket OAuth connection status
//	@Description	Returns whether a Bitbucket OAuth credential is stored. Only administrators can perform this action.
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	StatusResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Router			/oauth/bitbucket/status [get]
//
// status reports the stored Bitbucket OAuth connection (admin only).
func (h *Handler) status(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	token, err := h.oauthSvc.GetToken(c.Context(), user.ID)
	if errors.Is(err, oauth.ErrNotFound) {
		return c.JSON(StatusResponse{
			Connected:   false,
			ConnectedAt: nil,
			ExpiresAt:   nil,
			Scopes:      nil,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to get oauth status: %w", err)
	}

	return c.JSON(NewStatusResponse(token))
}

//	@Summary		Disconnect Bitbucket OAuth
//	@Description	Removes the stored Bitbucket OAuth credential. Remote repository webhooks are NOT removed; they stop working after the token expires. Only administrators can perform this action.
//	@Tags			OAuth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Router			/oauth/bitbucket/disconnect [post]
//
// disconnect removes the stored OAuth credential (admin only).
func (h *Handler) disconnect(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.ErrUnauthorized
	}

	if err := h.oauthSvc.DeleteToken(c.Context(), user.ID); err != nil {
		return fmt.Errorf("failed to disconnect bitbucket oauth: %w", err)
	}

	h.logger.Info("bitbucket oauth disconnected")

	return c.SendStatus(fiber.StatusNoContent)
}

// redirect sends the browser to the admin settings page with the given query
// string. Errors are wrapped per the repository error policy.
func (h *Handler) redirect(c *fiber.Ctx, query string) error {
	if err := c.Redirect("/" + query); err != nil {
		return fmt.Errorf("failed to redirect to admin settings: %w", err)
	}

	return nil
}
