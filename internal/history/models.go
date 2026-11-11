package history

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type historyModel struct {
	bun.BaseModel `bun:"table:history,alias:h"`

	ID          int64         `bun:"id,pk,autoincrement"`
	SubjectType string        `bun:"subject_type,notnull"`
	SubjectID   int64         `bun:"subject_id,notnull"`
	ActorID     int64         `bun:"actor_id,notnull"`
	Action      Action        `bun:"action,notnull"`
	Changes     []FieldChange `bun:"type:json,notnull"`
	CreatedAt   time.Time     `bun:"created_at,notnull"`
}

func newHistoryModel(
	subjectType string,
	subjectID, actorID int64,
	action Action,
	changes []FieldChange,
	createdAt time.Time,
) *historyModel {
	if changes == nil {
		changes = make([]FieldChange, 0)
	}
	return &historyModel{
		BaseModel:   schema.BaseModel{},
		ID:          0,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		ActorID:     actorID,
		Action:      action,
		Changes:     changes,
		CreatedAt:   createdAt,
	}
}

func (m *historyModel) toDomain() *Entry {
	if m == nil {
		return nil
	}

	changes := make([]FieldChange, len(m.Changes))
	copy(changes, m.Changes)

	return &Entry{
		ID: m.ID,
		EntryBase: EntryBase{
			SubjectType: m.SubjectType,
			SubjectID:   m.SubjectID,
			ActorID:     m.ActorID,
			Action:      m.Action,
			Changes:     changes,
		},
		CreatedAt: m.CreatedAt,
	}
}
