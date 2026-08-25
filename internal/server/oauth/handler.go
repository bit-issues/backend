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

// settingsPath is the frontend admin settings page hosting the OAuth
// connection card. The callback redirects there after the OAuth flow.
const settingsPath = "/admin"

// Handler serves the Bitbucket OAuth connection lifecycle endpoints.
type Handler struct {
	svc    *oauth.Service
	client *oauth.BitbucketClient
	logger *zap.Logger
}

// NewHandler creates the OAuth HTTP handler.
func NewHandler(svc *oauth.Service, client *oauth.BitbucketClient, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, client: client, logger: logger}
}

// RegisterPublic mounts the callback route. It must be registered before the
// JWT middleware: Bitbucket redirects the browser there, so the endpoint is
// public and authenticity is guaranteed by the CSRF state.
func (h *Handler) RegisterPublic(r fiber.Router) {
	r.Get("/oauth/bitbucket/callback", h.callback)
}

// Register mounts the admin-only OAuth lifecycle routes. The JWT middleware
// must already be applied to the router.
func (h *Handler) Register(r fiber.Router) {
	group := r.Group("/oauth/bitbucket")
	group.Get("/authorize", jwtauth.WithRole(users.RoleAdmin), h.authorize)
	group.Get("/status", jwtauth.WithRole(users.RoleAdmin), h.status)
	group.Post("/disconnect", jwtauth.WithRole(users.RoleAdmin), h.disconnect)
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

	if !h.client.Configured() {
		return fiber.NewError(fiber.StatusInternalServerError, "bitbucket oauth is not configured")
	}

	state, err := h.svc.CreateState(c.Context(), user.ID, h.client.RedirectURI())
	if err != nil {
		return fmt.Errorf("failed to create oauth state: %w", err)
	}

	return c.JSON(AuthorizeResponse{URL: h.client.AuthorizeURL(state)})
}

//	@Summary		Bitbucket OAuth callback
//	@Description	Public callback that consumes the CSRF state, exchanges the authorization code with Bitbucket, verifies the 'webhook' scope, stores the tokens, and redirects to the admin settings page.
//	@Tags			OAuth
//	@Param			code	query	string	false	"Authorization code"
//	@Param			state	query	string	false	"CSRF state"
//	@Param			error	query	string	false	"Bitbucket OAuth error code"
//	@Success		302
//	@Router			/oauth/bitbucket/callback [get]
//
// callback completes the Bitbucket OAuth connection flow (public). All
// outcomes are browser redirects; tokens are never included in them.
func (h *Handler) callback(c *fiber.Ctx) error {
	state := c.Query("state")
	code := c.Query("code")

	// Bitbucket declined the authorization (e.g. the admin denied consent).
	if denied := c.Query("error"); denied != "" {
		h.consumeStateBestEffort(c, state)
		return h.redirect(c, "?oauth=error&reason=access_denied")
	}

	if state == "" || code == "" {
		// Consume a supplied state even when the params are incomplete so a
		// dangling state can never be replayed against a later callback.
		h.consumeStateBestEffort(c, state)
		return h.redirect(c, "?oauth=error&reason=missing_params")
	}

	userID, err := h.svc.ConsumeStateForExchange(c.Context(), state, h.client.RedirectURI())
	if err != nil {
		h.logger.Warn("oauth callback rejected CSRF state", zap.Error(err))
		return h.redirect(c, "?oauth=error&reason=invalid_state")
	}

	token, err := h.client.Exchange(c.Context(), code)
	if err != nil {
		h.logger.Error("oauth callback token exchange failed", zap.Error(err))
		return h.redirect(c, "?oauth=error&reason=exchange_failed")
	}

	token.ConnectedByUserID = userID
	if saveErr := h.svc.SaveTokens(c.Context(), token); saveErr != nil {
		h.logger.Error("oauth callback failed to persist tokens", zap.Error(saveErr))
		reason := "save_failed"
		if errors.Is(saveErr, oauth.ErrInvalidScope) {
			reason = "invalid_scope"
		}
		return h.redirect(c, "?oauth=error&reason="+reason)
	}

	h.logger.Info("bitbucket oauth connected", zap.Int64("user_id", userID))

	return h.redirect(c, "?oauth=success")
}

// redirect sends the browser to the admin settings page with the given query
// string. Errors are wrapped per the repository error policy.
func (h *Handler) redirect(c *fiber.Ctx, query string) error {
	if err := c.Redirect(settingsPath + query); err != nil {
		return fmt.Errorf("failed to redirect to admin settings: %w", err)
	}

	return nil
}

// consumeStateBestEffort consumes a presented state even when the OAuth flow
// failed so a state can never be replayed. Failures are logged, never fatal.
func (h *Handler) consumeStateBestEffort(c *fiber.Ctx, state string) {
	if state == "" {
		return
	}

	if _, err := h.svc.ConsumeStateForExchange(c.Context(), state, h.client.RedirectURI()); err != nil {
		h.logger.Warn("oauth callback failed to consume state", zap.Error(err))
	}
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
	token, err := h.svc.GetStoredToken(c.Context())
	if err != nil {
		if errors.Is(err, oauth.ErrOAuthNotConnected) {
			return c.JSON(StatusResponse{
				Connected:   false,
				ConnectedAt: nil,
				ExpiresAt:   nil,
				Scopes:      nil,
			})
		}
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
//	@Success		200
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Router			/oauth/bitbucket/disconnect [post]
//
// disconnect removes the stored OAuth credential (admin only).
func (h *Handler) disconnect(c *fiber.Ctx) error {
	if err := h.svc.DeleteTokens(c.Context()); err != nil {
		return fmt.Errorf("failed to disconnect bitbucket oauth: %w", err)
	}

	h.logger.Info("bitbucket oauth disconnected")

	return c.SendStatus(fiber.StatusOK)
}
