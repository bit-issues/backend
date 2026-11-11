package tasks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/server/middlewares/jwtauth"
	"github.com/bit-issues/backend/internal/users"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// attachmentInitUpload initializes a file upload for a task attachment.
//
//	@Summary		Initiate attachment upload
//	@Description	Initialize a new attachment upload for a task. Returns a pre-signed URL for uploading the file.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		int64					true	"Task ID"
//	@Param			request	body		AttachmentUploadRequest	true	"Upload details"
//	@Success		201		{object}	AttachmentUploadResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/attachments [post]
func (h *Handler) attachmentInitUpload(c *fiber.Ctx, req *AttachmentUploadRequest) error {
	taskID, err := strconv.ParseInt(c.Params("task_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid task ID")
	}

	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	result, err := h.attachmentsSvc.InitUpload(c.Context(), attachments.AttachmentInput{
		TaskID:     taskID,
		FileName:   req.FileName,
		SizeBytes:  req.SizeBytes,
		UploaderID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize upload: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(toUploadResponse(result))
}

// attachmentConfirmUpload confirms an attachment upload after the file has been uploaded to the storage.
//
//	@Summary		Confirm attachment upload
//	@Description	Mark an attachment upload as completed after the file has been uploaded to the storage provider. Returns attachment metadata and download URL.
//	@Tags			Tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		int64	true	"Task ID"
//	@Param			id		path		int64	true	"Attachment ID"
//	@Success		200		{object}	AttachmentConfirmResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/attachments/{id}/confirm [put]
func (h *Handler) attachmentConfirmUpload(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid attachment ID")
	}

	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	attachment, err := h.attachmentsSvc.ConfirmUpload(c.Context(), id, user.ID)
	if err != nil {
		return fmt.Errorf("failed to confirm attachment upload: %w", err)
	}

	downloadURL, err := h.attachmentsSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to create download url: %w", err)
	}

	// Fetch user for uploaded_by enrichment (best effort; confirm already succeeded)
	usersMap, err := h.fetchUsersForAttachments(c.Context(), []attachments.Attachment{*attachment})
	if err != nil {
		h.logger.Error("failed to fetch users for attachments", zap.Int64("attachment_id", id), zap.Error(err))
		usersMap = make(map[int64]users.User)
	}

	return c.JSON(toConfirmResponse(attachment, downloadURL, usersMap))
}

// attachmentGetDownloadURL returns a download URL for an attachment.
//
//	@Summary		Get attachment download URL
//	@Description	Get a pre-signed download URL for a task attachment.
//	@Tags			Tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			task_id	path		int64	true	"Task ID"
//	@Param			id		path		int64	true	"Attachment ID"
//	@Success		200		{object}	AttachmentDownloadResponse
//	@Failure		400		{object}	fiberfx.ErrorResponse
//	@Failure		401		{object}	fiberfx.ErrorResponse
//	@Failure		404		{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/attachments/{id}/download [get]
func (h *Handler) attachmentGetDownloadURL(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid attachment ID")
	}

	if _, ok := jwtauth.GetUser(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	downloadURL, err := h.attachmentsSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get download url: %w", err)
	}

	return c.JSON(AttachmentDownloadResponse{DownloadURL: downloadURL})
}

// attachmentDelete deletes an attachment.
//
//	@Summary		Delete attachment
//	@Description	Delete a task attachment permanently.
//	@Tags			Tasks
//	@Security		BearerAuth
//	@Param			task_id	path	int64	true	"Task ID"
//	@Param			id		path	int64	true	"Attachment ID"
//	@Success		204
//	@Failure		400	{object}	fiberfx.ErrorResponse
//	@Failure		401	{object}	fiberfx.ErrorResponse
//	@Failure		404	{object}	fiberfx.ErrorResponse
//	@Router			/tasks/{task_id}/attachments/{id} [delete]
func (h *Handler) attachmentDelete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid attachment ID")
	}

	user, ok := jwtauth.GetUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if delErr := h.attachmentsSvc.Delete(c.Context(), user, id); delErr != nil {
		return fmt.Errorf("failed to delete attachment: %w", delErr)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// fetchUsersForAttachments collects unique user IDs from attachments and fetches them in a single batch call.
func (h *Handler) fetchUsersForAttachments(
	ctx context.Context,
	items []attachments.Attachment,
) (map[int64]users.User, error) {
	idSet := make(map[int64]struct{})
	for _, a := range items {
		idSet[a.UploadedBy] = struct{}{}
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
