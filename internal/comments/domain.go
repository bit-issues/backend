package comments

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxContentLength is the maximum length of comment content.
	MaxContentLength = 10000
)

// Comment represents a complete comment entity with all fields.
type Comment struct {
	ID        int64
	TaskID    int64
	AuthorID  int64
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// CommentInput contains the data required to create a new comment.
type CommentInput struct {
	TaskID   int64
	AuthorID int64
	Content  string
}

// CommentUpdate represents the data that can be updated for a comment.
type CommentUpdate struct {
	Content string
}

// Validate checks that the input data is valid for comment creation.
func (i CommentInput) Validate() error {
	// Validate task ID
	if i.TaskID <= 0 {
		return fmt.Errorf("%w: task_id must be positive", ErrValidationFailed)
	}

	// Validate author ID
	if i.AuthorID <= 0 {
		return fmt.Errorf("%w: author_id must be positive", ErrValidationFailed)
	}

	// Validate content
	content := strings.TrimSpace(i.Content)
	if content == "" {
		return fmt.Errorf("%w: content is required", ErrValidationFailed)
	}

	if utf8.RuneCountInString(content) > MaxContentLength {
		return fmt.Errorf("%w: content too long (max %d characters)", ErrValidationFailed, MaxContentLength)
	}

	return nil
}

// Validate checks that the update data is valid.
func (u CommentUpdate) Validate() error {
	// Validate content
	content := strings.TrimSpace(u.Content)
	if content == "" {
		return fmt.Errorf("%w: content is required", ErrValidationFailed)
	}

	if utf8.RuneCountInString(content) > MaxContentLength {
		return fmt.Errorf("%w: content too long (max %d characters)", ErrValidationFailed, MaxContentLength)
	}

	return nil
}
