package projects

import (
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/bit-issues/backend/internal/webhooks"
	restkit "github.com/capcom6/go-restkit"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for project-related endpoints.
type Handler struct {
	handler.Base

	projectsSvc *projects.Service
	webhooksSvc *webhooks.ManagementService
	usersSvc    *users.Service
	jwtSvc      *jwt.Service
}

// NewHandler creates a new Handler instance with the given dependencies.
func NewHandler(
	projectsSvc *projects.Service,
	webhooksSvc *webhooks.ManagementService,
	usersSvc *users.Service,
	jwtSvc *jwt.Service,
	validate *validator.Validate,
) handler.Handler {
	return &Handler{
		Base:        handler.Base{Validator: validate},
		projectsSvc: projectsSvc,
		webhooksSvc: webhooksSvc,
		usersSvc:    usersSvc,
		jwtSvc:      jwtSvc,
	}
}

// Register sets up the project routes on the given router.
// Routes are organized with appropriate middleware for authentication
// and authorization based on the operation.
func (h *Handler) Register(r fiber.Router) {
	// Public routes (authenticated users only)
	projects := r.Group(
		"/projects",
		h.errorsHandler,
	)

	projects.Get("/", h.list)
	projects.Get("/:slug", h.get)

	// Admin-only routes (require admin role)
	projects.Post("/",
		jwtauth.WithRole(users.RoleAdmin),
		validation.DecorateWithBodyEx(h.Validator, h.post),
	)
	projects.Patch("/:slug",
		jwtauth.WithRole(users.RoleAdmin),
		validation.DecorateWithBodyEx(h.Validator, h.patch),
	)
	projects.Delete("/:slug",
		jwtauth.WithRole(users.RoleAdmin),
		h.delete,
	)
	projects.Get("/:slug/webhook",
		jwtauth.WithRole(users.RoleAdmin),
		h.getWebhookStatus,
	)
	projects.Post("/:slug/webhook/register",
		jwtauth.WithRole(users.RoleAdmin),
		h.registerWebhook,
	)
	projects.Post("/:slug/webhook/unregister",
		jwtauth.WithRole(users.RoleAdmin),
		h.unregisterWebhook,
	)
}

//	@Summary		List all projects
//	@Description	Returns a paginated list of projects accessible to the authenticated user.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int		false	"Page limit (max 100)"	default(20)
//	@Param			offset	query		int		false	"Page offset"			default(0)
//	@Param			search	query		string	false	"Search by project name or slug (case-insensitive)"
//	@Success		200		{object}	ProjectListResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Router			/projects [get]
//
// list retrieves a paginated list of all projects.
func (h *Handler) list(c *fiber.Ctx) error {
	query := dto.PaginationQuery{
		Limit:  0,
		Offset: 0,
	}
	if err := h.QueryParserValidator(c, &query); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	// Parse search parameter
	search := c.Query("search", "")

	// Fetch projects from service
	projectsList, total, err := h.projectsSvc.List(
		c.Context(),
		projects.NewPagination(query.Limit, query.Offset),
		search,
	)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	return c.JSON(NewProjectListResponse(projectsList, total))
}

//	@Summary		Get project by ID
//	@Description	Returns detailed information about a specific project.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	ProjectResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug} [get]
//
// get retrieves a single project by its slug.
func (h *Handler) get(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Fetch project from service
	project, err := h.projectsSvc.GetBySlug(c.Context(), slug)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	return c.JSON(NewProjectResponse(project))
}

//	@Summary		Create a new project
//	@Description	Creates a new project linked to a BitBucket repository. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ProjectRequest	true	"Project creation data"
//	@Success		201		{object}	ProjectResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		409		{object}	fiberfx.ErrorResponse
//	@Router			/projects [post]
//
// post creates a new project (admin only).
func (h *Handler) post(c *fiber.Ctx, req *ProjectRequest) error {
	// Convert DTO to domain input
	input := req.toProjectInput()

	// Create project via service
	project, err := h.projectsSvc.Create(c.Context(), input)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(NewProjectResponse(project))
}

//	@Summary		Update a project
//	@Description	Updates project details. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string					true	"Project ID"
//	@Param			request	body		ProjectUpdateRequest	true	"Project update data"
//	@Success		200		{object}	ProjectResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		409		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug} [patch]
//
// patch updates an existing project (admin only).
func (h *Handler) patch(c *fiber.Ctx, req *ProjectUpdateRequest) error {
	slug := c.Params("slug")

	// Convert DTO to domain update
	update := req.toProjectUpdate()

	// Update project via service
	project, err := h.projectsSvc.Update(c.Context(), slug, update)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return c.JSON(NewProjectResponse(project))
}

//	@Summary		Delete a project
//	@Description	Permanently deletes a project and all its associated data. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path	string	true	"Project ID"
//	@Success		204
//	@Failure		400	{object}	fiberfx.ErrorResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug} [delete]
//
// delete removes a project (admin only).
func (h *Handler) delete(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Delete project via service
	if delErr := h.projectsSvc.Delete(c.Context(), slug); delErr != nil {
		return fmt.Errorf("failed to delete project: %w", delErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

//	@Summary		Get project webhook status
//	@Description	Returns the live Bitbucket webhook registration state of the project repository. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	WebhookStatusResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		502		{object}	fiberfx.ErrorResponse
//	@Failure		503		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook [get]
//
// getWebhookStatus returns the live Bitbucket webhook state (admin only).
func (h *Handler) getWebhookStatus(c *fiber.Ctx) error {
	project, err := h.resolveProject(c)
	if err != nil {
		return err
	}

	status, err := h.webhooksSvc.GetWebhookStatus(c.Context(), project)
	if err != nil {
		return fmt.Errorf("failed to get webhook status: %w", err)
	}

	return c.JSON(NewWebhookStatusResponse(status))
}

//	@Summary		Register project webhook
//	@Description	Registers or repairs the Bitbucket push webhook of the project repository. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	WebhookStatusResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		422		{object}	fiberfx.ErrorResponse
//	@Failure		502		{object}	fiberfx.ErrorResponse
//	@Failure		503		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook/register [post]
//
// registerWebhook registers or repairs the project webhook (admin only).
func (h *Handler) registerWebhook(c *fiber.Ctx) error {
	project, err := h.resolveProject(c)
	if err != nil {
		return err
	}

	status, err := h.webhooksSvc.RegisterWebhook(c.Context(), project)
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}

	return c.JSON(NewWebhookStatusResponse(status))
}

//	@Summary		Unregister project webhook
//	@Description	Removes the Bitbucket push webhook of the project repository. Only administrators can perform this action.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project ID"
//	@Success		200		{object}	WebhookStatusResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Failure		502		{object}	fiberfx.ErrorResponse
//	@Failure		503		{object}	fiberfx.ErrorResponse
//	@Router			/projects/{slug}/webhook/unregister [post]
//
// unregisterWebhook removes the project webhook (admin only).
func (h *Handler) unregisterWebhook(c *fiber.Ctx) error {
	project, err := h.resolveProject(c)
	if err != nil {
		return err
	}

	status, err := h.webhooksSvc.UnregisterWebhook(c.Context(), project)
	if err != nil {
		return fmt.Errorf("failed to unregister webhook: %w", err)
	}

	return c.JSON(NewWebhookStatusResponse(status))
}

// resolveProject loads the project referenced by the slug route parameter.
func (h *Handler) resolveProject(c *fiber.Ctx) (*projects.Project, error) {
	project, err := h.projectsSvc.GetBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// errorsHandler is a middleware that converts service errors to appropriate HTTP responses.
// It should be registered as the first middleware in the route group.
func (h *Handler) errorsHandler(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, projects.ErrValidationFailed):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, projects.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, projects.ErrNameAlreadyUsed):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, projects.ErrInvalidURL):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, webhooks.ErrWebhookSecretNotConfigured):
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	case restkit.IsClientError(err):
		code, reason := webhookClientErrorReason(err)
		return fiber.NewError(code, reason)
	case restkit.IsServerError(err):
		return fiber.NewError(fiber.StatusBadGateway, "Bitbucket API returned an error")
	case restkit.IsInfrastructureError(err):
		return fiber.NewError(fiber.StatusServiceUnavailable, "Bitbucket API is unreachable")
	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}

// webhookClientErrorReason maps a Bitbucket 4xx failure to a status code and
// a human-readable reason. The reason never exposes Bitbucket internals, the
// access token, or the webhook secret.
func webhookClientErrorReason(err error) (int, string) {
	code := fiber.StatusBadRequest
	reason := "Bitbucket rejected the webhook request"

	apiErr, ok := restkit.AsAPIError(err)
	if !ok {
		return code, reason
	}

	switch apiErr.StatusCode {
	case fiber.StatusBadRequest:
		reason = "Bitbucket rejected the webhook configuration"
	case fiber.StatusForbidden:
		code = fiber.StatusUnprocessableEntity
		reason = "insufficient Bitbucket permissions to manage webhooks"
	}

	return code, reason
}
