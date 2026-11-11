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
	subjectType string,
	subjectID int64,
	limit, offset int,
) ([]Entry, int, error) {
	return s.history.List(ctx, subjectType, subjectID, limit, offset)
}

// Record appends an entry base to the activity stream inside the caller's
// transaction, stamping CreatedAt itself.
func (s *Service) Record(ctx context.Context, tx bun.IDB, base EntryBase) error {
	return s.history.RecordWithin(ctx, tx, entryFromBase(base, time.Now()))
}
