package tasks

import (
	"strings"
	"time"

	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/comments"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	clientgo "github.com/bit-issues/client-go"
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
	Search     *string `query:"search"     validate:"omitempty,max=255"`
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
		Search:         q.Search,

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
	Kind        string  `json:"kind,omitempty"        validate:"omitempty,oneof=Bug Enhancement Task Proposal"`
	AssigneeID  *int64  `json:"assignee_id,omitempty" validate:"omitempty,min=1"`
	DueDate     *string `json:"due_date,omitempty"    validate:"omitempty,datetime=2006-01-02"`
}

// toTaskCreateInput converts a TaskCreateRequest DTO to a domain TaskInput.
func (req TaskCreateRequest) toTaskInput(authorID int64) tasks.TaskInput {
	priority := tasks.Priority(req.Priority)
	kind := tasks.Kind(req.Kind)

	return tasks.TaskInput{
		ProjectSlug: req.ProjectSlug,
		Title:       req.Title,
		Description: req.Description,
		Priority:    lo.CoalesceOrEmpty(priority, tasks.PriorityMinor),
		Status:      tasks.StatusNew, // API always creates as "New"
		Kind:        lo.CoalesceOrEmpty(kind, tasks.KindTask),
		AuthorID:    authorID,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
	}
}

// TaskUpdateRequest represents the request body for updating a task.
// All fields are optional to support partial updates.
//
//	@Description	Task update request with optional fields for partial updates.
type TaskUpdateRequest struct {
	Title       *string `json:"title,omitempty"       validate:"omitempty,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=10000"`
	Priority    *string `json:"priority,omitempty"    validate:"omitempty,oneof=Trivial Minor Major Critical Blocker"                                                default:"Minor"`
	Status      *string `json:"status,omitempty"      validate:"omitempty,oneof=New Open 'In Progress' Resolved Closed Reopened Invalid Duplicate Wontfix 'On Hold'"`
	Kind        *string `json:"kind,omitempty"        validate:"omitempty,oneof=Bug Enhancement Task Proposal"                                                       default:"Task"`
	AssigneeID  *int64  `json:"assignee_id,omitempty" validate:"omitempty,min=0"`
	DueDate     *string `json:"due_date,omitempty"    validate:"omitzero,datetime=2006-01-02"`
	Comment     *string `json:"comment,omitempty"     validate:"omitempty,max=10000"`
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

	var kind *tasks.Kind
	if req.Kind != nil {
		k := tasks.Kind(*req.Kind)
		kind = &k
	}

	return tasks.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		Priority:    priority,
		Status:      status,
		Kind:        kind,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
	}
}

// taskDetailsJSON keeps the established REST key order (comments before
// attachments) while carrying the shared clientgo wire payload. The shared
// TaskDetailsResponse declares attachments first; serialization order changes
// the response bytes, so the handler marshals through this backend wrapper.
type taskDetailsJSON struct {
	clientgo.TaskResponse

	Project     *clientgo.Project             `json:"project"`
	Comments    []clientgo.CommentResponse    `json:"comments"`
	Attachments []clientgo.AttachmentResponse `json:"attachments"`
}

// newTaskResponse converts a domain Task to the shared clientgo wire response.
// It enriches author and assignee fields from the pre-fetched users map.
// If a user is not found in the map, only the ID is populated (stub).
func newTaskResponse(task *tasks.Task, usersMap map[int64]users.User) clientgo.TaskResponse {
	return clientgo.TaskResponse{
		ID:          task.ID,
		ProjectSlug: task.ProjectSlug,
		Number:      task.Number,
		Title:       task.Title,
		Description: task.Description,
		Priority:    string(task.Priority),
		Status:      string(task.Status),
		Kind:        string(task.Kind),
		Author:      *dto.ResolveUserBrief(&task.AuthorID, usersMap),
		Assignee:    dto.ResolveUserBrief(task.AssigneeID, usersMap),
		DueDate:     task.DueDate,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	}
}

func newTaskDetailsResponse(
	task *tasks.Task,
	usersMap map[int64]users.User,
	comments []comments.Comment,
	attachmentList []attachments.AttachmentWithURL,
	project *clientgo.Project,
) *clientgo.TaskDetailsResponse {
	return &clientgo.TaskDetailsResponse{
		TaskResponse: newTaskResponse(task, usersMap),
		Project:      project,
		Comments:     toCommentsList(comments, usersMap),
		Attachments:  toAttachmentsList(attachmentList, usersMap),
	}
}

func taskDetailsJSONFrom(response *clientgo.TaskDetailsResponse) *taskDetailsJSON {
	return &taskDetailsJSON{
		TaskResponse: response.TaskResponse,
		Project:      response.Project,
		Comments:     response.Comments,
		Attachments:  response.Attachments,
	}
}

// toTaskListResponse converts a list of domain Tasks to the shared wire response.
func toTaskListResponse(items []tasks.Task, usersMap map[int64]users.User, total int) *clientgo.TaskListResponse {
	return &clientgo.TaskListResponse{
		Items: lo.Map(
			items,
			func(t tasks.Task, _ int) clientgo.TaskResponse {
				return newTaskResponse(&t, usersMap)
			},
		),
		Total: total,
	}
}
