package tasks

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type historyModel struct {
	bun.BaseModel `bun:"table:task_history,alias:th"`

	ID        int64         `bun:"id,pk,autoincrement"`
	TaskID    int64         `bun:"task_id,notnull"`
	ActorID   int64         `bun:"actor_id,notnull"`
	Action    HistoryAction `bun:"action,notnull"`
	Changes   []FieldChange `bun:"type:json,notnull"`
	CreatedAt time.Time     `bun:"created_at,notnull"`
}

func newHistoryModel(
	taskID, actorID int64,
	action HistoryAction,
	changes []FieldChange,
	createdAt time.Time,
) *historyModel {
	if changes == nil {
		changes = make([]FieldChange, 0)
	}
	return &historyModel{
		BaseModel: schema.BaseModel{},
		ID:        0,
		TaskID:    taskID,
		ActorID:   actorID,
		Action:    action,
		Changes:   changes,
		CreatedAt: createdAt,
	}
}

func (m *historyModel) toDomain() *TaskHistoryEntry {
	if m == nil {
		return nil
	}

	changes := make([]FieldChange, len(m.Changes))
	copy(changes, m.Changes)

	return &TaskHistoryEntry{
		ID:        m.ID,
		TaskID:    m.TaskID,
		ActorID:   m.ActorID,
		Action:    m.Action,
		Changes:   changes,
		CreatedAt: m.CreatedAt,
	}
}
