package attachments

import (
	"fmt"
	"time"
)

const (
	DefaultMaxFileSizeBytes uint64 = 104857600
	MaxFileNameLength       int    = 255
)

type AttachmentStatus string

const (
	StatusPending  AttachmentStatus = "pending"
	StatusUploaded AttachmentStatus = "uploaded"
)

type Attachment struct {
	ID         int64
	TaskID     int64
	FileName   string
	StorageKey string
	SizeBytes  uint64
	Status     AttachmentStatus
	UploadedBy int64
	UploadedAt time.Time
	DeletedAt  *time.Time
}

type AttachmentInput struct {
	TaskID     int64
	FileName   string
	SizeBytes  uint64
	UploaderID int64
}

type UploadResult struct {
	Attachment *Attachment
	UploadURL  string
}

type AttachmentWithURL struct {
	Attachment

	DownloadURL string
}

func (i AttachmentInput) Validate(maxFileSize uint64) error {
	if i.TaskID <= 0 {
		return fmt.Errorf("%w: task_id must be positive", ErrValidationFailed)
	}

	if i.UploaderID <= 0 {
		return fmt.Errorf("%w: uploader_id must be positive", ErrValidationFailed)
	}

	fileName := sanitizeFileName(i.FileName)
	if fileName == "" {
		return fmt.Errorf("%w: file_name is required", ErrValidationFailed)
	}

	if len(fileName) > MaxFileNameLength {
		return fmt.Errorf("%w: file_name too long (max %d characters)", ErrValidationFailed, MaxFileNameLength)
	}

	if i.SizeBytes == 0 {
		return fmt.Errorf("%w: size_bytes must be positive", ErrValidationFailed)
	}

	if i.SizeBytes > maxFileSize {
		return ErrFileTooLarge
	}

	return nil
}
