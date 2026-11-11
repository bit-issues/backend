package history

import (
	"context"
	"time"
)

// DefaultLimit is the page size used when a caller asks for no limit.
const DefaultLimit = 20

// Service reads the activity history and appends events best-effort:
// writers save their domain data first, then call Record. The activity feed
// is disposable - a failed record is the caller's problem to log, never the
// request's.
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

// Record appends an entry base to the activity stream, stamping CreatedAt
// itself. Callers treat failures as best-effort: the domain write already
// succeeded.
func (s *Service) Record(ctx context.Context, base EntryBase) error {
	return s.history.Record(ctx, entryFromBase(base, time.Now()))
}
