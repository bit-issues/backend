package comments

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
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

// SortOrder controls comment ordering. Fields are prefixed with "-" for descending order.
type SortOrder string

// Sort order constants.
const (
	SortOrderDefault       SortOrder = ""
	SortOrderCreatedAtDesc SortOrder = "-created_at"
	SortOrderCreatedAtAsc  SortOrder = "created_at"
)

func (s SortOrder) apply(query *bun.SelectQuery) *bun.SelectQuery {
	switch s {
	case SortOrderCreatedAtAsc:
		return query.OrderBy("created_at", bun.OrderAsc).OrderBy("id", bun.OrderAsc)
	case SortOrderDefault, SortOrderCreatedAtDesc:
		return query.OrderBy("created_at", bun.OrderDesc).OrderBy("id", bun.OrderDesc)
	default:
		return query.OrderBy("id", bun.OrderDesc)
	}
}

// CommentFilter contains filtering criteria for querying comments.
type CommentFilter struct {
	TaskID   *int64
	AuthorID *int64
}

func (f CommentFilter) apply(query *bun.SelectQuery) *bun.SelectQuery {
	if f.TaskID != nil && *f.TaskID > 0 {
		query = query.Where("task_id = ?", *f.TaskID)
	}
	if f.AuthorID != nil && *f.AuthorID > 0 {
		query = query.Where("author_id = ?", *f.AuthorID)
	}
	return query
}

type pagination struct {
}

// DefaultLimit implements [db.DefaultPagination].
func (p *pagination) DefaultLimit() int {
	return DefaultLimit
}

// MaxLimit implements [db.DefaultPagination].
func (p *pagination) MaxLimit() int {
	return MaxLimit
}

var _ db.DefaultPagination = (*pagination)(nil)

type Pagination = db.Pagination[*pagination]

func NewPagination(limit, offset int) *Pagination {
	return db.NewPagination[*pagination](limit, offset)
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
