package history

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// DefaultLimit is the page size used when a caller asks for no limit.
const DefaultLimit = 20

// Service reads the activity history and appends events inside the writing
// module's transaction: callers start their own transaction, write their data,
// then pass the transaction here so event and trigger commit atomically.
type Service struct {
	history *Repository
}

// NewService creates a new Service instance with the given dependencies.
func NewService(history *Repository) *Service {
	return &Service{history: history}
}

// List retrieves history entries for a subject newest-first.
func (s *Service) List(
	ctx context.Context,
	subjectType SubjectType,
	subjectID int64,
	limit, offset int,
) ([]Entry, int, error) {
	return s.history.List(ctx, subjectType, subjectID, limit, offset)
}

// RecordCreated records a task creation inside the caller's transaction.
func (s *Service) RecordCreated(
	ctx context.Context,
	tx bun.IDB,
	taskID, actorID int64,
	createdAt time.Time,
) error {
	return s.history.RecordWithin(ctx, tx, newCreatedEntry(taskID, actorID, createdAt))
}

// RecordUpdated records applied field changes inside the caller's transaction.
func (s *Service) RecordUpdated(
	ctx context.Context,
	tx bun.IDB,
	taskID, actorID int64,
	changes []FieldChange,
	createdAt time.Time,
) error {
	return s.history.RecordWithin(ctx, tx, newUpdatedEntry(taskID, actorID, changes, createdAt))
}

// RecordAttachmentAdded records a confirmed upload inside the caller's transaction.
func (s *Service) RecordAttachmentAdded(
	ctx context.Context,
	tx bun.IDB,
	taskID, actorID int64,
	fileName string,
	createdAt time.Time,
) error {
	return s.history.RecordWithin(ctx, tx, newAttachmentAddedEntry(taskID, actorID, fileName, createdAt))
}

// RecordAttachmentRemoved records a deletion inside the caller's transaction.
func (s *Service) RecordAttachmentRemoved(
	ctx context.Context,
	tx bun.IDB,
	taskID, actorID int64,
	fileName string,
	createdAt time.Time,
) error {
	return s.history.RecordWithin(ctx, tx, newAttachmentRemovedEntry(taskID, actorID, fileName, createdAt))
}
