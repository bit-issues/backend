package projects

import (
	"errors"
	"fmt"

	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for project-related endpoints.
type Handler struct {
	handler.Base

	projectsSvc *projects.Service
	usersSvc    *users.Service
	jwtSvc      *jwt.Service
}

// NewHandler creates a new Handler instance with the given dependencies.
func NewHandler(
	projectsSvc *projects.Service,
	usersSvc *users.Service,
	jwtSvc *jwt.Service,
	validate *validator.Validate,
) handler.Handler {
	return &Handler{
		Base:        handler.Base{Validator: validate},
		projectsSvc: projectsSvc,
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
}

//	@Summary		List all projects
//	@Description	Returns a paginated list of projects accessible to the authenticated user.
//	@Tags			Projects
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Page limit (max 100)"	default(20)
//	@Param			offset	query		int	false	"Page offset"			default(0)
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

	// Fetch projects from service
	projectsList, err := h.projectsSvc.List(c.Context(), query.Limit, query.Offset)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Get total count
	total, err := h.projectsSvc.Count(c.Context())
	if err != nil {
		return fmt.Errorf("failed to count projects: %w", err)
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
	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}
