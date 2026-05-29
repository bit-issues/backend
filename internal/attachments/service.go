package attachments

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	config Config

	attachments *Repository

	tasksSvc   *tasks.Service
	storageSvc *storage.Service

	logger *zap.Logger
}

func NewService(
	config Config,
	attachments *Repository,
	tasksSvc *tasks.Service,
	storageSvc *storage.Service,
	logger *zap.Logger,
) *Service {
	if config.MaxSize == 0 {
		config.MaxSize = DefaultMaxFileSizeBytes
	}

	return &Service{
		config: config,

		attachments: attachments,

		tasksSvc:   tasksSvc,
		storageSvc: storageSvc,

		logger: logger,
	}
}

func (s *Service) InitUpload(ctx context.Context, input AttachmentInput) (*UploadResult, error) {
	if err := input.Validate(s.config.MaxSize); err != nil {
		return nil, err
	}

	ok, err := s.tasksSvc.Exists(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if task exists: %w", err)
	}
	if !ok {
		return nil, ErrTaskNotFound
	}

	storageKey := s.buildStorageKey(input.TaskID, input.FileName)
	uploadURL, err := s.storageSvc.PresignedPutObject(ctx, storageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload url: %w", err)
	}

	attachment, err := s.attachments.Create(ctx, input, storageKey)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Attachment: attachment,
		UploadURL:  uploadURL,
	}, nil
}

func (s *Service) Import(
	ctx context.Context,
	taskID int64,
	fileName, localFilePath string,
	uploadedBy int64,
) (*Attachment, error) {
	fi, err := os.Stat(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat attachment file: %w", err)
	}

	storageKey := s.buildStorageKey(taskID, fileName)

	if putErr := s.storageSvc.PutObject(ctx, storageKey, localFilePath); putErr != nil {
		return nil, fmt.Errorf("failed to upload attachment to storage: %w", putErr)
	}

	//nolint:gosec // file size is always non-negative
	attachment, createErr := s.attachments.Import(
		ctx,
		newAttachmentImport(taskID, fileName, storageKey, uint64(fi.Size()), uploadedBy),
	)
	if createErr != nil {
		if cleanupErr := s.storageSvc.Delete(ctx, storageKey); cleanupErr != nil {
			s.logger.Warn(
				"failed to cleanup storage after failed attachment import",
				zap.String("storageKey", storageKey),
				zap.Error(cleanupErr),
			)
		}
		return nil, createErr
	}

	return attachment, nil
}

func (s *Service) ListByTask(ctx context.Context, taskID int64) ([]AttachmentWithURL, error) {
	items, err := s.attachments.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	result := make([]AttachmentWithURL, 0, len(items))
	for _, item := range items {
		downloadURL, urlErr := s.storageSvc.PresignedGetObject(ctx, item.StorageKey)
		if urlErr != nil {
			return nil, fmt.Errorf("failed to create download url: %w", urlErr)
		}

		result = append(result, AttachmentWithURL{Attachment: item, DownloadURL: downloadURL})
	}

	return result, nil
}

func (s *Service) GetDownloadURL(ctx context.Context, id int64) (string, error) {
	attachment, err := s.attachments.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	if attachment.Status != StatusUploaded {
		return "", ErrNotUploaded
	}

	downloadURL, err := s.storageSvc.PresignedGetObject(ctx, attachment.StorageKey)
	if err != nil {
		return "", fmt.Errorf("failed to create download url: %w", err)
	}

	return downloadURL, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, id int64, uploaderID int64) (*Attachment, error) {
	attachment, err := s.attachments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if attachment.UploadedBy != uploaderID {
		return nil, ErrUnauthorized
	}

	if confirmErr := s.attachments.Confirm(ctx, id); confirmErr != nil {
		return nil, confirmErr
	}

	attachment.Status = StatusUploaded
	return attachment, nil
}

func (s *Service) Delete(ctx context.Context, user *users.User, id int64) error {
	attachment, err := s.attachments.GetByID(ctx, id)
	if err != nil {
		return err
	}

	task, err := s.tasksSvc.GetByID(ctx, attachment.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if user.Role != users.RoleAdmin && user.ID != attachment.UploadedBy && user.ID != task.AuthorID {
		return ErrUnauthorized
	}

	if delErr := s.attachments.Delete(ctx, id); delErr != nil {
		return delErr
	}

	if delErr := s.storageSvc.Delete(ctx, attachment.StorageKey); delErr != nil {
		return fmt.Errorf("attachment metadata deleted, but object cleanup failed: %w", delErr)
	}

	return nil
}

func (s *Service) buildStorageKey(taskID int64, fileName string) string {
	safeName := sanitizeFileName(fileName)
	return path.Join(strconv.FormatInt(taskID, 10), uuid.Must(uuid.NewV7()).String()+"-"+safeName)
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	base := filepath.Base(trimmed)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	return strings.ReplaceAll(base, "\\", "")
}
