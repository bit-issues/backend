package tasks

import (
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/server/dto"
)

type AttachmentUploadRequest struct {
	FileName  string `json:"file_name"  validate:"required"`
	SizeBytes uint64 `json:"size_bytes" validate:"required,min=1"`
}

type AttachmentUploadResponse struct {
	ID        int64  `json:"id"`
	FileName  string `json:"file_name"`
	SizeBytes uint64 `json:"size_bytes"`
	UploadURL string `json:"upload_url"`
}

type AttachmentConfirmResponse struct {
	ID          int64         `json:"id"`
	FileName    string        `json:"file_name"`
	SizeBytes   uint64        `json:"size_bytes"`
	DownloadURL string        `json:"download_url"`
	UploadedAt  string        `json:"uploaded_at"`
	UploadedBy  dto.UserBrief `json:"uploaded_by"`
}

type AttachmentDownloadResponse struct {
	DownloadURL string `json:"download_url"`
}

type AttachmentResponse struct {
	ID          int64         `json:"id"`
	FileName    string        `json:"file_name"`
	SizeBytes   uint64        `json:"size_bytes"`
	UploadedBy  dto.UserBrief `json:"uploaded_by"`
	UploadedAt  string        `json:"uploaded_at"`
	DownloadURL string        `json:"download_url"`
}

func toUploadResponse(result *attachments.UploadResult) AttachmentUploadResponse {
	return AttachmentUploadResponse{
		ID:        result.Attachment.ID,
		FileName:  result.Attachment.FileName,
		SizeBytes: result.Attachment.SizeBytes,
		UploadURL: result.UploadURL,
	}
}

func toConfirmResponse(attachment *attachments.Attachment, downloadURL string) AttachmentConfirmResponse {
	return AttachmentConfirmResponse{
		ID:          attachment.ID,
		FileName:    attachment.FileName,
		SizeBytes:   attachment.SizeBytes,
		DownloadURL: downloadURL,
		UploadedAt:  attachment.UploadedAt.UTC().Format(time.RFC3339),
		UploadedBy: dto.UserBrief{
			ID:        attachment.UploadedBy,
			Email:     "",
			Role:      "",
			CreatedAt: "",
		},
	}
}

func toAttachmentResponse(item attachments.AttachmentWithURL) AttachmentResponse {
	return AttachmentResponse{
		ID:          item.ID,
		FileName:    item.FileName,
		SizeBytes:   item.SizeBytes,
		UploadedAt:  item.UploadedAt.UTC().Format(time.RFC3339),
		DownloadURL: item.DownloadURL,
		UploadedBy: dto.UserBrief{
			ID:        item.UploadedBy,
			Email:     "",
			Role:      "",
			CreatedAt: "",
		},
	}
}

func toAttachmentsList(items []attachments.AttachmentWithURL) []AttachmentResponse {
	result := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toAttachmentResponse(item))
	}

	return result
}
