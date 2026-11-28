package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
)

func toUploadResponse(result *attachments.UploadResult) clientgo.AttachmentUploadResponse {
	return clientgo.AttachmentUploadResponse{
		ID:        result.Attachment.ID,
		FileName:  result.Attachment.FileName,
		SizeBytes: result.Attachment.SizeBytes,
		UploadURL: result.UploadURL,
	}
}

func toConfirmResponse(
	attachment *attachments.Attachment,
	downloadURL string,
	usersMap map[int64]users.User,
) clientgo.AttachmentResponse {
	return clientgo.AttachmentResponse{
		ID:          attachment.ID,
		FileName:    attachment.FileName,
		SizeBytes:   attachment.SizeBytes,
		DownloadURL: downloadURL,
		UploadedAt:  attachment.UploadedAt.UTC().Format(time.RFC3339),
		UploadedBy:  *dto.ResolveUserBrief(&attachment.UploadedBy, usersMap),
	}
}

func toAttachmentResponse(
	item attachments.AttachmentWithURL,
	usersMap map[int64]users.User,
) clientgo.AttachmentResponse {
	return clientgo.AttachmentResponse{
		ID:          item.ID,
		FileName:    item.FileName,
		SizeBytes:   item.SizeBytes,
		UploadedAt:  item.UploadedAt.UTC().Format(time.RFC3339),
		DownloadURL: item.DownloadURL,
		UploadedBy:  *dto.ResolveUserBrief(&item.UploadedBy, usersMap),
	}
}

func toAttachmentsList(
	items []attachments.AttachmentWithURL,
	usersMap map[int64]users.User,
) []clientgo.AttachmentResponse {
	result := make([]clientgo.AttachmentResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toAttachmentResponse(item, usersMap))
	}

	return result
}
