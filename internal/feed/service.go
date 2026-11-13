package feed

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bit-issues/backend/internal/history"
	"github.com/bit-issues/backend/internal/tasks"
	"github.com/bit-issues/backend/internal/users"
	"go.uber.org/zap"
)

// Service reads the task activity feed from history, derives field-level
// change pairs at read time, and enriches the page with the users needed to
// render it. The feed is a disposable view: a missing context row or a failed
// user lookup degrades rendering, never the domain data.
type Service struct {
	historySvc *history.Service
	usersSvc   *users.Service
	logger     *zap.Logger
}

// NewService creates a new Service instance with the given dependencies.
func NewService(historySvc *history.Service, usersSvc *users.Service, logger *zap.Logger) *Service {
	return &Service{historySvc: historySvc, usersSvc: usersSvc, logger: logger}
}

// List returns one page of task activity entries newest-first with their
// derived field changes (values rendered for display: assignee ids resolved
// to names), the users rendering the page needs, and the total entry count.
// It fetches one extra context row to seed change derivation for the boundary
// entry and drops it before responding; payloads of every fetched row (page
// plus context row) cover both sides of every derived assignee pair, so
// AssigneeIDs is collected from all of them. A failed user lookup is logged
// and the page degrades to an empty Users map.
func (s *Service) List(ctx context.Context, taskID int64, offset, limit int) (Page, error) {
	if _, ok := feedWindow(offset, limit); !ok {
		return Page{}, ErrWindowTooLarge
	}

	fetchedList, total, err := s.historySvc.List(
		ctx,
		tasks.SubjectTypeTask,
		taskID,
		limit+1,
		offset,
	)
	if err != nil {
		return Page{}, fmt.Errorf("failed to list task history: %w", err)
	}

	entries := fetchedList
	changes := deriveHistoryChanges(fetchedList)
	if len(fetchedList) > limit {
		entries = fetchedList[:limit]
		changes = changes[:limit]
	}
	assigneeIDs := collectAssigneeIDs(fetchedList)
	usersMap := s.lookupUsers(ctx, taskID, entries, assigneeIDs)

	return Page{
		Entries:     entries,
		Changes:     resolveChangeValues(changes, usersMap),
		AssigneeIDs: assigneeIDs,
		Users:       usersMap,
		Total:       total,
	}, nil
}

// lookupUsers batch-fetches the users behind the page: every entry's actor
// plus both sides of every derived assignee pair (already collected across
// the fetched window). Failures are logged and degrade to an empty map -
// actors fall back to placeholders in the DTO layer.
func (s *Service) lookupUsers(
	ctx context.Context,
	taskID int64,
	entries []history.Entry,
	assigneeIDs []int64,
) map[int64]users.User {
	idSet := make(map[int64]struct{})

	for i := range entries {
		idSet[entries[i].ActorID] = struct{}{}
	}
	for _, id := range assigneeIDs {
		idSet[id] = struct{}{}
	}

	if len(idSet) == 0 {
		return make(map[int64]users.User)
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	usersMap, err := s.usersSvc.LookupByIDs(ctx, ids)
	if err != nil {
		s.logger.Error("failed to fetch users for feed",
			zap.Int64("task_id", taskID),
			zap.Error(err),
		)
		return make(map[int64]users.User)
	}

	return usersMap
}

// collectAssigneeIDs parses the assignee payload values of every fetched
// entry into user IDs.
func collectAssigneeIDs(entries []history.Entry) []int64 {
	idSet := make(map[int64]struct{})

	for i := range entries {
		collectAssigneeID(&idSet, entries[i].Payload[tasks.FieldAssignee])
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids
}

func collectAssigneeID(idSet *map[int64]struct{}, value *string) {
	if value == nil {
		return
	}

	id, err := strconv.ParseInt(*value, 10, 64)
	if err != nil {
		return
	}

	(*idSet)[id] = struct{}{}
}
