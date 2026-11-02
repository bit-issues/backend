package tasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

//	@Summary		Get task activity feed
//	@Description	Returns comments and field-level task changes merged newest-first.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Project slug"
//	@Param			number	path		int		true	"Task number within project"
//	@Param			limit	query		int		false	"Page limit (max 100)"	default(20)
//	@Param			offset	query		int		false	"Page offset"			default(0)
//	@Success		200		{object}	FeedResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{slug}/{number}/feed [get]
//
// feed retrieves the merged activity feed for a task.
func (h *Handler) feed(c *fiber.Ctx) error {
	slug := c.Params("slug")

	number, err := strconv.ParseInt(c.Params("number"), 10, 64)
	if err != nil || number < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task number")
	}

	query := new(FeedQuery)
	if parseErr := h.QueryParserValidator(c, query); parseErr != nil {
		return fmt.Errorf("failed to parse query: %w", parseErr)
	}

	task, err := h.tasksSvc.GetByProjectAndNumber(c.Context(), slug, int(number))
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	window := query.Offset + query.Limit
	commentList, commentTotal, err := h.commentsSvc.ListByTaskPage(c.Context(), task.ID, window, 0)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	historyList, historyTotal, err := h.tasksSvc.ListHistory(c.Context(), task.ID, window, 0)
	if err != nil {
		return fmt.Errorf("failed to list task history: %w", err)
	}

	entries := pageFeedEntries(mergeFeedEntries(commentList, historyList), query.Offset, query.Limit)

	usersMap, err := h.fetchUsersForFeed(c.Context(), entries)
	if err != nil {
		h.logger.Error("failed to fetch users for feed", zap.Int64("task_id", task.ID), zap.Error(err))
		usersMap = make(map[int64]users.User)
	}

	return c.JSON(toFeedResponse(entries, usersMap, commentTotal+historyTotal))
}

func (h *Handler) fetchUsersForFeed(
	ctx context.Context,
	entries []feedEntry,
) (map[int64]users.User, error) {
	idSet := make(map[int64]struct{})

	for i := range entries {
		entry := &entries[i]

		if entry.Comment != nil {
			idSet[entry.Comment.AuthorID] = struct{}{}
		}

		if entry.History == nil {
			continue
		}

		idSet[entry.History.ActorID] = struct{}{}
		for _, change := range entry.History.Changes {
			if change.Field != tasks.HistoryFieldAssignee {
				continue
			}

			collectHistoryAssignee(&idSet, change.OldValue)
			collectHistoryAssignee(&idSet, change.NewValue)
		}
	}

	if len(idSet) == 0 {
		return make(map[int64]users.User), nil
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	usersMap, err := h.usersSvc.LookupByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	return usersMap, nil
}

func collectHistoryAssignee(idSet *map[int64]struct{}, value *string) {
	if value == nil {
		return
	}

	id, err := strconv.ParseInt(*value, 10, 64)
	if err != nil {
		return
	}

	(*idSet)[id] = struct{}{}
}
