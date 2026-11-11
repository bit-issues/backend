package tasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

//	@Summary		Get task activity feed
//	@Description	Returns comments, field-level task changes and attachment events merged newest-first.
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
// feed retrieves the merged activity feed for a task.
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

	window, ok := feedWindow(query.Offset, query.Limit)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "offset is too large")
	}

	commentList, commentTotal, err := h.commentsSvc.List(
		c.Context(),
		comments.CommentFilter{TaskID: &task.ID, AuthorID: nil},
		comments.SortOrderDefault,
		comments.NewPagination(window, 0),
	)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	// One extra history row seeds read-side change derivation for the
	// boundary entry; it is dropped before merging.
	historyList, historyTotal, err := h.historySvc.List(c.Context(), tasks.SubjectTypeTask, task.ID, window+1, 0)
	if err != nil {
		return fmt.Errorf("failed to list task history: %w", err)
	}

	allHistory := historyList
	historyChanges := deriveHistoryChanges(historyList)
	if len(historyList) > window {
		historyList = historyList[:window]
		historyChanges = historyChanges[:window]
	}

	entries := pageFeedEntries(mergeFeedEntries(commentList, historyList, historyChanges), query.Offset, query.Limit)

	usersMap, err := h.fetchUsersForFeed(c.Context(), entries, allHistory)
	if err != nil {
		h.logger.Error("failed to fetch users for feed", zap.Int64("task_id", task.ID), zap.Error(err))
		usersMap = make(map[int64]users.User)
	}

	return c.JSON(toFeedResponse(entries, usersMap, commentTotal+historyTotal))
}

func (h *Handler) fetchUsersForFeed(
	ctx context.Context,
	entries []feedEntry,
	historyList []history.Entry,
) (map[int64]users.User, error) {
	idSet := make(map[int64]struct{})

	for i := range entries {
		entry := &entries[i]

		if entry.Comment != nil {
			idSet[entry.Comment.AuthorID] = struct{}{}
		}

		if entry.History != nil {
			idSet[entry.History.ActorID] = struct{}{}
		}
	}

	// Payloads of every fetched history row (including the dropped context
	// row) cover both sides of every derived assignee pair.
	for i := range historyList {
		collectHistoryAssignee(&idSet, historyList[i].Payload[tasks.FieldAssignee])
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
