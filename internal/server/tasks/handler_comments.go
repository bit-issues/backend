package tasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

//	@Summary		Create a new comment on a task
//	@Description	Adds a new comment to the specified task.
//	@Tags			Comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		int64					true	"Task ID"
//	@Param			request	body		CommentCreateRequest	true	"Comment creation data"
//	@Success		201		{object}	CommentResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/comments [post]
//
// createComment creates a new comment on a task.
func (h *Handler) createComment(c *fiber.Ctx, req *CommentCreateRequest) error {
	// Extract task ID from URL params
	taskID, err := strconv.ParseInt(c.Params("task_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Get current user from JWT context
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Create comment via service
	comment, err := h.commentsSvc.Create(c.Context(), comments.CommentInput{
		TaskID:   taskID,
		AuthorID: user.ID,
		Content:  req.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	// Fetch user for author enrichment (best-effort)
	usersMap, err := h.fetchUsersForComments(c.Context(), []comments.Comment{*comment})
	if err != nil {
		h.logger.Error("failed to fetch users for comment", zap.Int64("comment_id", comment.ID), zap.Error(err))
		usersMap = make(map[int64]users.User)
	}

	return c.Status(fiber.StatusCreated).JSON(toCommentResponse(comment, usersMap))
}

//	@Summary		Update a comment
//	@Description	Updates the content of an existing comment. Only the comment author or admin can update.
//	@Tags			Comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		int64					true	"Task ID"
//	@Param			id		path		int64					true	"Comment ID"
//	@Param			request	body		CommentUpdateRequest	true	"Comment update data"
//	@Success		200		{object}	CommentResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		403		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/comments/{id} [put]
//
// updateComment modifies an existing comment.
func (h *Handler) updateComment(c *fiber.Ctx, req *CommentUpdateRequest) error {
	// Extract task ID from URL params
	taskID, err := strconv.ParseInt(c.Params("task_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Extract comment ID from URL params
	commentID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid comment ID")
	}

	// Get current user from JWT context
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Update comment via service
	comment, err := h.commentsSvc.Update(c.Context(), user.ID, taskID, commentID, comments.CommentUpdate{
		Content: req.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	// Fetch user for author enrichment (best-effort)
	usersMap, err := h.fetchUsersForComments(c.Context(), []comments.Comment{*comment})
	if err != nil {
		// Log error but continue with ID-only author stub
		// The comment was successfully updated, don't fail the request due to user lookup failure
		usersMap = make(map[int64]users.User)
	}

	return c.JSON(toCommentResponse(comment, usersMap))
}

//	@Summary		Delete a comment
//	@Description	Soft deletes a comment. Only the comment author or admin can delete.
//	@Tags			Comments
//	@Security		BearerAuth
//	@Param			task_id	path	int64	true	"Task ID"
//	@Param			id		path	int64	true	"Comment ID"
//	@Success		204
//	@Failure		400	{object}	fiberfx.ErrorResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		403	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/comments/{id} [delete]
//
// deleteComment soft-deletes a comment.
func (h *Handler) deleteComment(c *fiber.Ctx) error {
	// Extract task ID from URL params
	taskID, err := strconv.ParseInt(c.Params("task_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	// Extract comment ID from URL params
	commentID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid comment ID")
	}

	// Get current user from JWT context
	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Delete comment via service
	if delErr := h.commentsSvc.Delete(c.Context(), user.ID, taskID, commentID); delErr != nil {
		return fmt.Errorf("failed to delete comment: %w", delErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// fetchUsersForComments collects unique user IDs from comments and fetches them in a single batch call.
func (h *Handler) fetchUsersForComments(
	ctx context.Context,
	items []comments.Comment,
) (map[int64]users.User, error) {
	idSet := make(map[int64]struct{})
	for _, c := range items {
		idSet[c.AuthorID] = struct{}{}
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
