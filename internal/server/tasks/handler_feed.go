package tasks

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

//	@Summary		Get task activity feed
//	@Description	Returns field-level task changes and attachment events newest-first.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project slug"
//	@Param			number	path		int		true	"Task number within project"
//	@Param			limit	query		int		false	"Page limit (max 100)"								default(20)
//	@Param			offset	query		int		false	"Page offset (offset+limit must not exceed 5000)"	default(0)
//	@Success		200		{object}	FeedResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{slug}/{number}/feed [get]
//
// feed retrieves the activity feed for a task.
func (h *Handler) feed(c *fiber.Ctx) error {
	slug := c.Params("slug")

	number, err := strconv.ParseInt(c.Params("number"), 10, 64)
	if err != nil || number < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task number")
	}

	query := new(FeedQuery)
	query.Limit = 20
	if parseErr := h.QueryParserValidator(c, query); parseErr != nil {
		return fmt.Errorf("failed to parse query: %w", parseErr)
	}

	task, err := h.tasksSvc.GetByProjectAndNumber(c.Context(), slug, int(number))
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	page, err := h.feedSvc.List(c.Context(), task.ID, query.Offset, query.Limit)
	if err != nil {
		return fmt.Errorf("failed to build task feed: %w", err)
	}

	return c.JSON(toFeedResponse(page.Entries, page.Changes, page.Users, page.Total))
}
