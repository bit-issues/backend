package history

import "context"

// DefaultLimit is the page size used when a caller asks for no limit.
const DefaultLimit = 20

// Service exposes read access to the activity history.
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
