package comments

import (
	"context"
	"fmt"

	"github.com/bit-issues/backend/internal/users"
	"go.uber.org/zap"
)

const (
	// DefaultLimit is the default number of comments to return per page.
	DefaultLimit = 20
	// MaxLimit is the maximum number of comments to return per page.
	MaxLimit = 100
)

// Service implements the business logic for comment management.
type Service struct {
	comments *Repository

	usersSvc *users.Service

	logger *zap.Logger
}

// NewService creates a new Service instance with the given dependencies.
func NewService(comments *Repository, usersSvc *users.Service, logger *zap.Logger) *Service {
	return &Service{
		comments: comments,

		usersSvc: usersSvc,

		logger: logger,
	}
}

// Create validates input and creates a new comment.
func (s *Service) Create(ctx context.Context, input CommentInput) (*Comment, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Create comment
	return s.comments.Create(ctx, input)
}

// Import creates a comment with explicit timestamps for import.
func (s *Service) Import(
	ctx context.Context,
	input Comment,
) (*Comment, error) {
	// Create comment with import-specific data
	return s.comments.Import(ctx, input)
}

// GetByID retrieves a comment by its ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Comment, error) {
	return s.comments.GetByID(ctx, id)
}

// ListByTask retrieves all comments for a specific task.
func (s *Service) ListByTask(ctx context.Context, taskID int64) ([]Comment, error) {
	return s.comments.List(ctx, taskID)
}

// Update modifies an existing comment with the provided content.
// Returns an error if the comment is not found, validation fails, or the user is not authorized.
func (s *Service) Update(ctx context.Context, userID int64, taskID, id int64, update CommentUpdate) (*Comment, error) {
	// Validate update content
	if err := update.Validate(); err != nil {
		return nil, err
	}

	// Fetch existing comment
	comment, err := s.comments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if comment.TaskID != taskID {
		return nil, ErrUnauthorized
	}

	// Check authorization
	if authErr := s.validateAuthor(ctx, userID, comment.AuthorID); authErr != nil {
		return nil, authErr
	}

	// Perform update
	if updErr := s.comments.Update(ctx, id, update.Content); updErr != nil {
		return nil, updErr
	}

	// Return the updated comment
	return s.comments.GetByID(ctx, id)
}

// Delete soft-deletes a comment.
// Returns an error if the comment is not found, validation fails, or the user is not authorized.
func (s *Service) Delete(ctx context.Context, userID int64, taskID, id int64) error {
	// Fetch existing comment
	comment, err := s.comments.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.TaskID != taskID {
		return ErrUnauthorized
	}

	// Check authorization
	if authErr := s.validateAuthor(ctx, userID, comment.AuthorID); authErr != nil {
		return authErr
	}

	// Perform soft delete
	return s.comments.Delete(ctx, id)
}

func (s *Service) validateAuthor(ctx context.Context, userID int64, authorID int64) error {
	if userID != authorID {
		if isAdmin, err := s.usersSvc.IsAdmin(ctx, userID); err != nil {
			return fmt.Errorf("failed to check admin status: %w", err)
		} else if !isAdmin {
			return ErrUnauthorized
		}
	}

	return nil
}
