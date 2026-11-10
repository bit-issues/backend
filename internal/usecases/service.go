package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"github.com/uptrace/bun"
)

// Service coordinates cross-module flows that commit atomically.
type Service struct {
	db              *bun.DB
	tasksRepo       *tasks.Repository
	historyRepo     *history.Repository
	attachmentsRepo *attachments.Repository
	projectsSvc     *projects.Service
	storageSvc      *storage.Service
}

// NewService creates a new Service instance with the given dependencies.
func NewService(
	db *bun.DB,
	tasksRepo *tasks.Repository,
	historyRepo *history.Repository,
	attachmentsRepo *attachments.Repository,
	projectsSvc *projects.Service,
	storageSvc *storage.Service,
) *Service {
	return &Service{
		db:              db,
		tasksRepo:       tasksRepo,
		historyRepo:     historyRepo,
		attachmentsRepo: attachmentsRepo,
		projectsSvc:     projectsSvc,
		storageSvc:      storageSvc,
	}
}

// CreateTask validates the input and creates the task together with its
// created history entry in one transaction.
func (s *Service) CreateTask(ctx context.Context, input tasks.TaskInput) (*tasks.Task, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task input: %w", err)
	}

	exists, err := s.projectsSvc.Exists(ctx, input.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to verify project: %w", err)
	}
	if !exists {
		return nil, tasks.ErrProjectNotFound
	}

	var task *tasks.Task
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		created, createErr := s.tasksRepo.CreateWithin(ctx, tx, input)
		if createErr != nil {
			return fmt.Errorf("transaction: %w", createErr)
		}
		task = created

		if recordErr := s.historyRepo.RecordWithin(
			ctx, tx,
			history.NewCreatedEntry(created.ID, created.AuthorID, created.CreatedAt),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// ImportTask creates a task with explicit metadata together with its created
// history entry in one transaction.
func (s *Service) ImportTask(ctx context.Context, input tasks.Task) (*tasks.Task, error) {
	var task *tasks.Task
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		imported, importErr := s.tasksRepo.ImportWithin(ctx, tx, input)
		if importErr != nil {
			return fmt.Errorf("transaction: %w", importErr)
		}
		task = imported

		if recordErr := s.historyRepo.RecordWithin(
			ctx, tx,
			history.NewCreatedEntry(imported.ID, imported.AuthorID, imported.CreatedAt),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to import task: %w", err)
	}

	return task, nil
}

// UpdateTask applies a task update together with its updated history entry in
// one transaction. A no-op update records no entry.
func (s *Service) UpdateTask(
	ctx context.Context,
	id int64,
	update tasks.TaskUpdate,
	actorID int64,
) (*tasks.Task, error) {
	if update.IsEmpty() {
		return nil, fmt.Errorf("%w: no update data provided", tasks.ErrValidationFailed)
	}

	if err := update.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task update: %w", err)
	}

	if actorID <= 0 {
		return nil, fmt.Errorf("%w: actor_id must be positive", tasks.ErrValidationFailed)
	}

	var updated *tasks.Task
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		task, changes, updateErr := s.tasksRepo.UpdateWithin(ctx, tx, id, update)
		if updateErr != nil {
			return fmt.Errorf("transaction: %w", updateErr)
		}
		updated = task

		if len(changes) == 0 {
			return nil
		}

		if recordErr := s.historyRepo.RecordWithin(
			ctx, tx,
			history.NewUpdatedEntry(id, actorID, changes, time.Now()),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return updated, nil
}

// ConfirmAttachment marks a pending upload complete together with its
// attachment-added history entry in one transaction. Re-confirming an already
// uploaded attachment is a no-op.
func (s *Service) ConfirmAttachment(ctx context.Context, id, uploaderID int64) (*attachments.Attachment, error) {
	attachment, err := s.attachmentsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	if attachment.UploadedBy != uploaderID {
		return nil, attachments.ErrUnauthorized
	}

	if attachment.Status == attachments.StatusUploaded {
		return attachment, nil
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if confirmErr := s.attachmentsRepo.ConfirmWithin(ctx, tx, id); confirmErr != nil {
			return fmt.Errorf("transaction: %w", confirmErr)
		}

		if recordErr := s.historyRepo.RecordWithin(
			ctx, tx,
			history.NewAttachmentAddedEntry(attachment.TaskID, uploaderID, attachment.FileName, time.Now()),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to confirm attachment: %w", err)
	}

	attachment.Status = attachments.StatusUploaded
	return attachment, nil
}

// DeleteAttachment removes an attachment row together with its
// attachment-removed history entry in one transaction, then cleans up the
// stored object.
func (s *Service) DeleteAttachment(ctx context.Context, user *users.User, id int64) error {
	attachment, err := s.attachmentsRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get attachment: %w", err)
	}

	task, err := s.tasksRepo.GetByID(ctx, attachment.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if user.Role != users.RoleAdmin && user.ID != attachment.UploadedBy && user.ID != task.AuthorID {
		return attachments.ErrUnauthorized
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if delErr := s.attachmentsRepo.DeleteWithin(ctx, tx, id); delErr != nil {
			return fmt.Errorf("transaction: %w", delErr)
		}

		if recordErr := s.historyRepo.RecordWithin(
			ctx, tx,
			history.NewAttachmentRemovedEntry(attachment.TaskID, user.ID, attachment.FileName, time.Now()),
		); recordErr != nil {
			return fmt.Errorf("transaction: %w", recordErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if delErr := s.storageSvc.Delete(ctx, attachment.StorageKey); delErr != nil {
		return fmt.Errorf("attachment metadata deleted, but object cleanup failed: %w", delErr)
	}

	return nil
}
