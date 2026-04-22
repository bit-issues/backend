package tasks

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for task-related endpoints.
type Handler struct {
	handler.Base

	tasksSvc       *tasks.Service
	commentsSvc    *comments.Service
	attachmentsSvc *attachments.Service
	usersSvc       *users.Service
}

// NewHandler creates a new Handler instance with the given dependencies.
func NewHandler(
	tasksSvc *tasks.Service,
	commentsSvc *comments.Service,
	attachmentsSvc *attachments.Service,
	usersSvc *users.Service,
	validate *validator.Validate,
) handler.Handler {
	return &Handler{
		Base: handler.Base{Validator: validate},

		tasksSvc:       tasksSvc,
		commentsSvc:    commentsSvc,
		attachmentsSvc: attachmentsSvc,
		usersSvc:       usersSvc,
	}
}

// Register sets up the task routes on the given router.
// Routes are organized with appropriate middleware for authentication
// and authorization based on the operation.
func (h *Handler) Register(r fiber.Router) {
	// Public routes (authenticated users only)
	tasks := r.Group(
		"/tasks",
		h.errorsHandler,
	)

	tasks.Get("/", h.list)
	tasks.Get("/me", h.myTasks)
	tasks.Get("/:id", h.get)
	tasks.Post("/",
		validation.DecorateWithBodyEx(h.Validator, h.post),
	)
	tasks.Patch("/:id",
		validation.DecorateWithBodyEx(h.Validator, h.patch),
	)
	tasks.Delete("/:id", h.delete)

	comments := tasks.Group("/:task_id/comments")
	comments.Post("/",
		validation.DecorateWithBodyEx(h.Validator, h.createComment),
	)
	comments.Put("/:id",
		validation.DecorateWithBodyEx(h.Validator, h.updateComment),
	)
	comments.Delete("/:id", h.deleteComment)

	attachments := tasks.Group("/:task_id/attachments")
	attachments.Post("/", validation.DecorateWithBodyEx(h.Validator, h.attachmentInitUpload))
	attachments.Put("/:id/confirm", h.attachmentConfirmUpload)
	attachments.Get("/:id/download", h.attachmentGetDownloadURL)
	attachments.Delete("/:id", h.attachmentDelete)
}

//	@Summary		List all tasks
//	@Description	Returns a paginated list of tasks with optional filtering.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit		query		int			false	"Page limit (max 100)"	default(20)
//	@Param			offset		query		int			false	"Page offset"			default(0)
//	@Param			project		query		string		false	"Filter by project slug"
//	@Param			author		query		int64		false	"Filter by author ID"
//	@Param			assignee	query		int64		false	"Filter by assignee ID"
//	@Param			statuses	query		[]string	false	"Filter by status (comma-separated)"
//	@Param			priorities	query		[]string	false	"Filter by priority (comma-separated)"
//	@Param			sort		query		string		false	"Sort field (e.g., created_at, -priority)"	default(created_at)
//	@Success		200			{object}	TaskListResponse
//	@Failure		401			{object}	fiberfx.ErrorResponse
//	@Router			/tasks [get]
//
// list retrieves a paginated list of tasks with optional filtering.
func (h *Handler) list(c *fiber.Ctx) error {
	query := new(TaskListQuery)
	if err := h.QueryParserValidator(c, query); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	filter := query.toFilter()

	// Fetch tasks from service
	taskList, total, err := h.tasksSvc.List(c.Context(), filter, query.Sort, query.toPagination())
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	return c.JSON(toTaskListResponse(taskList, total))
}

//	@Summary		Get task by ID
//	@Description	Returns detailed information about a specific task.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int64	true	"Task ID"
//	@Success		200	{object}	TaskDetailsResponse
//	@Failure		400	{object}	fiberfx.ErrorResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{id} [get]
//
// get retrieves a single task by its ID.
func (h *Handler) get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Fetch task from service
	task, err := h.tasksSvc.GetByID(c.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	comments, err := h.commentsSvc.ListByTask(c.Context(), task.ID)
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	attachmentList, err := h.attachmentsSvc.ListByTask(c.Context(), task.ID)
	if err != nil {
		return fmt.Errorf("failed to get attachments: %w", err)
	}

	return c.JSON(newTaskDetailsResponse(task, comments, attachmentList))
}

//	@Summary		Create a new task
//	@Description	Creates a new task in the specified project.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		TaskCreateRequest	true	"Task creation data"
//	@Success		201		{object}	TaskDetailsResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks [post]
//
// post creates a new task.
func (h *Handler) post(c *fiber.Ctx, req *TaskCreateRequest) error {
	// Get current user from JWT context
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Convert DTO to domain input
	input := req.toTaskInput(user.ID)

	// Create task via service
	task, err := h.tasksSvc.Create(c.Context(), input)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(newTaskDetailsResponse(task, nil, nil))
}

//	@Summary		Update a task
//	@Description	Updates task details.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int64				true	"Task ID"
//	@Param			request	body		TaskUpdateRequest	true	"Task update data"
//	@Success		200		{object}	TaskResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{id} [patch]
//
// patch updates an existing task.
func (h *Handler) patch(c *fiber.Ctx, req *TaskUpdateRequest) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Convert DTO to domain update
	update := req.toTaskUpdate()

	// Update task via service
	task, err := h.tasksSvc.Update(c.Context(), id, update)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return c.JSON(newTaskResponse(task))
}

//	@Summary		Delete a task
//	@Description	Soft-deletes a task (preserves it in the database for audit).
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	int64	true	"Task ID"
//	@Success		204
//	@Failure		400	{object}	fiberfx.ErrorResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{id} [delete]
//
// delete soft-deletes a task.
func (h *Handler) delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Delete task via service
	if delErr := h.tasksSvc.Delete(c.Context(), id); delErr != nil {
		return fmt.Errorf("failed to delete task: %w", delErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

//	@Summary		Get my tasks
//	@Description	Returns tasks assigned to or created by the authenticated user.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Page limit (max 100)"	default(20)
//	@Param			offset	query		int	false	"Page offset"			default(0)
//	@Success		200		{object}	TaskListResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/me [get]
//
// myTasks retrieves tasks assigned to or created by the current user.
func (h *Handler) myTasks(c *fiber.Ctx) error {
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	query := new(TaskListQuery)
	if err := h.QueryParserValidator(c, query); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	filter := query.toFilter()
	filter.UserID = &user.ID

	// Fetch tasks from service
	taskList, total, err := h.tasksSvc.List(c.Context(), filter, query.Sort, query.toPagination())
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	return c.JSON(toTaskListResponse(taskList, total))
}

// errorsHandler is a middleware that converts service errors to appropriate HTTP responses.
// It should be registered as the first middleware in the route group.
func (h *Handler) errorsHandler(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, tasks.ErrValidationFailed):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, tasks.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, tasks.ErrProjectNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())

	case errors.Is(err, comments.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, comments.ErrUnauthorized):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, comments.ErrValidationFailed):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())

	case errors.Is(err, attachments.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, attachments.ErrValidationFailed):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, attachments.ErrTaskNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, attachments.ErrUnauthorized):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, attachments.ErrFileTooLarge):
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, attachments.ErrNotUploaded):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())

	default:
		return err //nolint:wrapcheck // err is already wrapped
	}
}
