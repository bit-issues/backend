package tasks

import (
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/samber/lo"
)

// TaskListQuery represents pagination and sorting query parameters.
type TaskListQuery struct {
	dto.PaginationQuery
	dto.SortQuery

	Project    *string `query:"project"    validate:"omitempty,max=255"`
	Author     *int64  `query:"author"     validate:"omitempty,min=1"`
	Assignee   *int64  `query:"assignee"   validate:"omitempty,min=0"`
	Statuses   *string `query:"statuses"`
	Priorities *string `query:"priorities"`
}

func (q *TaskListQuery) toFilter() tasks.TaskFilter {
	statuses := []tasks.Status{}
	if q.Statuses != nil {
		statuses = lo.Map(
			strings.Split(*q.Statuses, ","),
			func(s string, _ int) tasks.Status { return tasks.Status(s) },
		)
	}
	priorities := []tasks.Priority{}
	if q.Priorities != nil {
		priorities = lo.Map(
			strings.Split(*q.Priorities, ","),
			func(s string, _ int) tasks.Priority { return tasks.Priority(s) },
		)
	}

	return tasks.TaskFilter{
		ProjectSlug:    q.Project,
		AuthorID:       q.Author,
		AssigneeID:     q.Assignee,
		Statuses:       statuses,
		Priorities:     priorities,
		IncludeDeleted: false,

		DueFrom:     nil,
		DueTo:       nil,
		CreatedFrom: nil,
		CreatedTo:   nil,

		UserID: nil,
	}
}

func (q *TaskListQuery) toPagination() *tasks.Pagination {
	return tasks.NewPagination(q.Limit, q.Offset)
}

// TaskCreateRequest represents the request body for creating a new task.
//
//	@Description	Task creation request with title, description, priority, assignee, and due date.
type TaskCreateRequest struct {
	ProjectSlug string  `json:"project_slug"          validate:"required,max=255"`
	Title       string  `json:"title"                 validate:"required,max=255"`
	Description string  `json:"description,omitempty" validate:"max=10000"`
	Priority    string  `json:"priority,omitempty"    validate:"omitempty,oneof=Trivial Minor Major Critical Blocker"`
	AssigneeID  *int64  `json:"assignee_id,omitempty" validate:"omitempty,min=1"`
	DueDate     *string `json:"due_date,omitempty"    validate:"omitempty,datetime=2006-01-02"`
}

// TaskUpdateRequest represents the request body for updating a task.
// All fields are optional to support partial updates.
//
//	@Description	Task update request with optional fields for partial updates.
type TaskUpdateRequest struct {
	Title       *string `json:"title,omitempty"       validate:"omitempty,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=10000"`
	Priority    *string `json:"priority,omitempty"    validate:"omitempty,oneof=Trivial Minor Major Critical Blocker"            default:"Minor"`
	Status      *string `json:"status,omitempty"      validate:"omitempty,oneof=New Open 'In Progress' Resolved Closed Reopened"`
	AssigneeID  *int64  `json:"assignee_id,omitempty" validate:"omitempty,min=0"`
	DueDate     *string `json:"due_date,omitempty"    validate:"omitzero,datetime=2006-01-02"`
}

// TaskResponse represents the API response for a single task.
//
//	@Description	Full task details with nested author and assignee information.
type TaskResponse struct {
	ID          int64          `json:"id"`
	ProjectSlug string         `json:"project_slug"`
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Priority    string         `json:"priority"`
	Status      string         `json:"status"`
	Author      dto.UserBrief  `json:"author"`
	Assignee    *dto.UserBrief `json:"assignee,omitempty"`
	DueDate     *string        `json:"due_date,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// newTaskResponse converts a domain Task to a TaskResponse DTO.
// It requires fetching author and assignee from the users service.
func newTaskResponse(task *tasks.Task) TaskResponse {
	var assignee *dto.UserBrief
	if task.AssigneeID != nil {
		assignee = &dto.UserBrief{
			ID:        *task.AssigneeID,
			Name:      "",
			Role:      "",
			CreatedAt: "",
		}
	}

	return TaskResponse{
		ID:          task.ID,
		ProjectSlug: task.ProjectSlug,
		Number:      task.Number,
		Title:       task.Title,
		Description: task.Description,
		Priority:    string(task.Priority),
		Status:      string(task.Status),
		Author: dto.UserBrief{
			ID:        task.AuthorID,
			Name:      "",
			Role:      "",
			CreatedAt: "",
		},
		Assignee:  assignee,
		DueDate:   task.DueDate,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
		UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
	}
}

type TaskDetailsResponse struct {
	TaskResponse

	Comments    []CommentResponse    `json:"comments"`
	Attachments []AttachmentResponse `json:"attachments"`
}

func newTaskDetailsResponse(
	task *tasks.Task,
	comments []comments.Comment,
	attachmentList []attachments.AttachmentWithURL,
) *TaskDetailsResponse {
	return &TaskDetailsResponse{
		TaskResponse: newTaskResponse(task),
		Comments:     toCommentsList(comments),
		Attachments:  toAttachmentsList(attachmentList),
	}
}

// TaskListResponse represents the API response for a list of tasks.
//
//	@Description	Paginated list of tasks with total count.
type TaskListResponse struct {
	Items []TaskResponse `json:"items"`
	Total int            `json:"total"`
}

// toTaskListResponse converts a list of domain Tasks to TaskListResponse DTO.
func toTaskListResponse(items []tasks.Task, total int) TaskListResponse {
	return TaskListResponse{
		Items: lo.Map(
			items,
			func(t tasks.Task, _ int) TaskResponse {
				return newTaskResponse(&t)
			},
		),
		Total: total,
	}
}

// toTaskUpdate converts a TaskUpdateRequest DTO to a domain TaskUpdate.
// The domain layer uses pointer-to-pointer types to distinguish between:
// - nil (field not provided/unchanged)
// - *nil (set to zero value/null)
// - *value (set to specific value).
func (req TaskUpdateRequest) toTaskUpdate() tasks.TaskUpdate {
	var priority *tasks.Priority
	if req.Priority != nil {
		p := tasks.Priority(*req.Priority)
		priority = &p
	}

	var status *tasks.Status
	if req.Status != nil {
		s := tasks.Status(*req.Status)
		status = &s
	}

	return tasks.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
		Status:      status,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
	}
}

// toTaskCreateInput converts a TaskCreateRequest DTO to a domain TaskInput.
func (req TaskCreateRequest) toTaskInput(authorID int64) tasks.TaskInput {
	priority := tasks.Priority(req.Priority)

	return tasks.TaskInput{
		ProjectSlug: req.ProjectSlug,
		Title:       req.Title,
		Description: req.Description,
		Priority:    lo.CoalesceOrEmpty(priority, tasks.PriorityMinor),
		AuthorID:    authorID,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
	}
}
