package attachments

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type attachmentModel struct {
	bun.BaseModel `bun:"table:attachments,alias:a"`

	ID         int64      `bun:"id,pk,autoincrement"`
	TaskID     int64      `bun:"task_id,notnull"`
	FileName   string     `bun:"file_name,notnull"`
	StorageKey string     `bun:"storage_key,notnull"`
	SizeBytes  uint64     `bun:"size_bytes,notnull"`
	Status     string     `bun:"status,notnull"`
	UploadedBy int64      `bun:"uploaded_by,notnull"`
	UploadedAt time.Time  `bun:"uploaded_at,notnull"`
	DeletedAt  *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

func newAttachmentModel(input AttachmentInput, storageKey string) *attachmentModel {
	return &attachmentModel{
		BaseModel: schema.BaseModel{},
		ID:        0,

		TaskID:     input.TaskID,
		FileName:   sanitizeFileName(input.FileName),
		StorageKey: storageKey,
		SizeBytes:  input.SizeBytes,
		Status:     string(StatusPending),
		UploadedBy: input.UploaderID,
		UploadedAt: time.Now().UTC(),
		DeletedAt:  nil,
	}
}

func newAttachmentImport(
	taskID int64,
	fileName, storageKey string,
	sizeBytes uint64,
	uploadedBy int64,
) *attachmentModel {
	return &attachmentModel{
		BaseModel: schema.BaseModel{},
		ID:        0,

		TaskID:     taskID,
		FileName:   sanitizeFileName(fileName),
		StorageKey: storageKey,
		SizeBytes:  sizeBytes,
		Status:     string(StatusUploaded),
		UploadedBy: uploadedBy,
		UploadedAt: time.Now().UTC(),
		DeletedAt:  nil,
	}
}

func (m *attachmentModel) toDomain() *Attachment {
	if m == nil {
		return nil
	}

	return &Attachment{
		ID:         m.ID,
		TaskID:     m.TaskID,
		FileName:   m.FileName,
		StorageKey: m.StorageKey,
		SizeBytes:  m.SizeBytes,
		Status:     AttachmentStatus(m.Status),
		UploadedBy: m.UploadedBy,
		UploadedAt: m.UploadedAt,
		DeletedAt:  m.DeletedAt,
	}
}
