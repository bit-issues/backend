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

type taskListQuery clientgo.TaskListQuery

func (q *taskListQuery) toFilter() tasks.TaskFilter {
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

func (q *taskListQuery) toPagination() *tasks.Pagination {
	return tasks.NewPagination(q.Limit, q.Offset)
}

type taskCreateRequest clientgo.TaskCreateRequest

// toTaskInput converts the shared task creation request to a domain TaskInput.
func (req *taskCreateRequest) toTaskInput(authorID int64) tasks.TaskInput {
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

type taskUpdateRequest clientgo.TaskUpdateRequest

// toTaskUpdate converts the shared task update request to a domain TaskUpdate.
// The domain layer uses pointer-to-pointer types to distinguish between:
// - nil (field not provided/unchanged)
// - *nil (set to zero value/null)
// - *value (set to specific value).
func (req *taskUpdateRequest) toTaskUpdate() tasks.TaskUpdate {
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
