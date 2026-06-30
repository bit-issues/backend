package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/db"
	"github.com/uptrace/bun"
)

const MaxTitleLength = 255

// Priority represents task priority levels, matching BitBucket values.
type Priority string

// Priority constants.
const (
	PriorityTrivial  Priority = "Trivial"
	PriorityMinor    Priority = "Minor"
	PriorityMajor    Priority = "Major"
	PriorityCritical Priority = "Critical"
	PriorityBlocker  Priority = "Blocker"
)

// IsValid checks if the priority value is one of the allowed constants.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityTrivial, PriorityMinor, PriorityMajor, PriorityCritical, PriorityBlocker:
		return true
	default:
		return false
	}
}

// Kind represents the type of task, matching BitBucket values.
type Kind string

// Kind constants.
const (
	KindBug         Kind = "Bug"
	KindEnhancement Kind = "Enhancement"
	KindTask        Kind = "Task"
	KindProposal    Kind = "Proposal"
)

// IsValid checks if the kind value is one of the allowed constants.
func (k Kind) IsValid() bool {
	switch k {
	case KindBug, KindEnhancement, KindTask, KindProposal:
		return true
	default:
		return false
	}
}

// String returns the string representation of Kind.
func (k Kind) String() string {
	return string(k)
}

// Status represents task lifecycle states, matching BitBucket values.
type Status string

// Status constants.
const (
	StatusNew        Status = "New"
	StatusOpen       Status = "Open"
	StatusInProgress Status = "In Progress"
	StatusResolved   Status = "Resolved"
	StatusClosed     Status = "Closed"
	StatusReopened   Status = "Reopened"
	StatusInvalid    Status = "Invalid"
	StatusDuplicate  Status = "Duplicate"
	StatusWontfix    Status = "Wontfix"
	StatusOnHold     Status = "On Hold"
)

// IsValid checks if the status value is one of the allowed constants.
func (s Status) IsValid() bool {
	switch s {
	case StatusNew, StatusOpen, StatusInProgress, StatusResolved, StatusClosed, StatusReopened,
		StatusInvalid, StatusDuplicate, StatusWontfix, StatusOnHold:
		return true
	default:
		return false
	}
}

// Task represents a complete task entity with all fields.
type Task struct {
	ID          int64
	ProjectSlug string
	Number      int
	Title       string
	Description string
	Priority    Priority
	Status      Status
	Kind        Kind
	AuthorID    int64
	AssigneeID  *int64
	DueDate     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// TaskInput contains the data required to create a new task.
type TaskInput struct {
	ProjectSlug string
	Title       string
	Description string
	Priority    Priority
	Status      Status
	Kind        Kind
	AuthorID    int64
	AssigneeID  *int64
	DueDate     *string // YYYY-MM-DD format
}

// Validate checks that the input data is valid for task creation.
func (i TaskInput) Validate() error {
	// Validate project slug
	projectSlug := strings.TrimSpace(i.ProjectSlug)
	if projectSlug == "" {
		return fmt.Errorf("%w: project slug is required", ErrValidationFailed)
	}

	// Validate title
	title := strings.TrimSpace(i.Title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrValidationFailed)
	}

	if len(title) > MaxTitleLength {
		return fmt.Errorf("%w: title too long (max 255 characters)", ErrValidationFailed)
	}

	// Validate priority
	if !i.Priority.IsValid() {
		return fmt.Errorf("%w: invalid priority value", ErrValidationFailed)
	}

	// Validate status
	if !i.Status.IsValid() {
		return fmt.Errorf("%w: invalid status value", ErrValidationFailed)
	}

	// Validate kind
	if !i.Kind.IsValid() {
		return fmt.Errorf("%w: invalid kind value", ErrValidationFailed)
	}

	// AuthorID must be positive
	if i.AuthorID <= 0 {
		return fmt.Errorf("%w: author_id must be positive", ErrValidationFailed)
	}

	// If assignee is provided, it must be positive
	if i.AssigneeID != nil && *i.AssigneeID <= 0 {
		return fmt.Errorf("%w: assignee_id must be positive", ErrValidationFailed)
	}

	if i.DueDate != nil {
		if _, err := time.Parse(time.DateOnly, *i.DueDate); err != nil {
			return fmt.Errorf("%w: invalid due_date format", ErrValidationFailed)
		}
	}

	return nil
}

// TaskUpdate represents the data that can be updated for a task.
// All fields are optional. Pointers are used to distinguish between
// "set to zero value" and "not provided".
type TaskUpdate struct {
	Title       *string
	Description *string
	Priority    *Priority
	Status      *Status
	Kind        *Kind
	AssigneeID  *int64  // nil=unchanged, 0=set to NULL, value=set to ID
	DueDate     *string // nil=unchanged, ""=set to NULL, value=set date string
}

// IsEmpty returns true if no update fields are set.
func (u TaskUpdate) IsEmpty() bool {
	return u.Title == nil &&
		u.Description == nil &&
		u.Priority == nil &&
		u.Status == nil &&
		u.Kind == nil &&
		u.AssigneeID == nil &&
		u.DueDate == nil
}

func (u TaskUpdate) Validate() error {
	if u.Title != nil {
		// Trim and validate title
		title := strings.TrimSpace(*u.Title)
		if title == "" {
			return fmt.Errorf("%w: title is required", ErrValidationFailed)
		}

		if len(title) > MaxTitleLength {
			return fmt.Errorf("%w: title too long (max 255 characters)", ErrValidationFailed)
		}
	}

	if u.Priority != nil && !u.Priority.IsValid() {
		return fmt.Errorf("%w: invalid priority value", ErrValidationFailed)
	}

	if u.Status != nil && !u.Status.IsValid() {
		return fmt.Errorf("%w: invalid status value", ErrValidationFailed)
	}

	if u.Kind != nil && !u.Kind.IsValid() {
		return fmt.Errorf("%w: invalid kind value", ErrValidationFailed)
	}

	if u.AssigneeID != nil && *u.AssigneeID < 0 {
		return fmt.Errorf("%w: assignee_id must be non-negative", ErrValidationFailed)
	}

	if u.DueDate != nil && *u.DueDate != "" {
		if _, err := time.Parse(time.DateOnly, *u.DueDate); err != nil {
			return fmt.Errorf("%w: invalid due_date format", ErrValidationFailed)
		}
	}

	return nil
}

// TaskFilter contains filtering criteria for querying tasks.
type TaskFilter struct {
	ProjectSlug    *string
	AuthorID       *int64
	AssigneeID     *int64
	Statuses       []Status
	Priorities     []Priority
	DueFrom        *time.Time
	DueTo          *time.Time
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	IncludeDeleted bool

	Search *string // Full-text search across title and description

	// Extended
	UserID *int64 // AuthorID or AssigneeID
}

func (f TaskFilter) apply(query *bun.SelectQuery) *bun.SelectQuery {
	query = f.applyBasicFilters(query)
	query = f.applyDateFilters(query)
	query = f.applySearchFilter(query)
	query = f.applyExtendedFilters(query)
	return query
}

func (f TaskFilter) applyBasicFilters(query *bun.SelectQuery) *bun.SelectQuery {
	if f.ProjectSlug != nil && *f.ProjectSlug != "" {
		query = query.Where("project_slug = ?", *f.ProjectSlug)
	}
	if f.AuthorID != nil && *f.AuthorID > 0 {
		query = query.Where("author_id = ?", *f.AuthorID)
	}
	if f.AssigneeID != nil {
		if *f.AssigneeID == 0 {
			query = query.Where("assignee_id IS NULL")
		} else {
			query = query.Where("assignee_id = ?", *f.AssigneeID)
		}
	}
	if len(f.Statuses) > 0 {
		query = query.Where("status IN (?)", bun.List(f.Statuses))
	}
	if len(f.Priorities) > 0 {
		query = query.Where("priority IN (?)", bun.List(f.Priorities))
	}
	return query
}

func (f TaskFilter) applyDateFilters(query *bun.SelectQuery) *bun.SelectQuery {
	if f.DueFrom != nil {
		query = query.Where("due_date >= ?", *f.DueFrom)
	}
	if f.DueTo != nil {
		query = query.Where("due_date <= ?", *f.DueTo)
	}
	if f.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		query = query.Where("created_at <= ?", *f.CreatedTo)
	}
	return query
}

func (f TaskFilter) applySearchFilter(query *bun.SelectQuery) *bun.SelectQuery {
	if f.Search != nil && *f.Search != "" {
		clean := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`+-<>()~*"@`, r) {
				return -1
			}
			return r
		}, *f.Search)
		clean = strings.TrimSpace(clean)
		if clean != "" {
			terms := strings.Fields(clean)
			for i, t := range terms {
				terms[i] = "+" + t + "*"
			}
			query = query.Where("MATCH(title, description) AGAINST(? IN BOOLEAN MODE)", strings.Join(terms, " "))
		}
	}
	return query
}

func (f TaskFilter) applyExtendedFilters(query *bun.SelectQuery) *bun.SelectQuery {
	if f.IncludeDeleted {
		query = query.WhereAllWithDeleted()
	}
	if f.UserID != nil {
		query = query.WhereGroup("AND", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("author_id = ?", *f.UserID).WhereOr("assignee_id = ?", *f.UserID)
		})
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
