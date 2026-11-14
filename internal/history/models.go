package history

import (
	"maps"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type historyModel struct {
	bun.BaseModel `bun:"table:history,alias:h"`

	ID          int64              `bun:"id,pk,autoincrement"`
	SubjectType string             `bun:"subject_type,notnull"`
	SubjectID   int64              `bun:"subject_id,notnull"`
	ActorID     int64              `bun:"actor_id,notnull"`
	Action      Action             `bun:"action,notnull"`
	Payload     map[string]*string `bun:"type:json,notnull"`
	CreatedAt   time.Time          `bun:"created_at,notnull"`
}

func newHistoryModel(
	subjectType string,
	subjectID, actorID int64,
	action Action,
	payload map[string]*string,
	createdAt time.Time,
) *historyModel {
	if payload == nil {
		payload = make(map[string]*string)
	}
	return &historyModel{
		BaseModel:   schema.BaseModel{},
		ID:          0,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		ActorID:     actorID,
		Action:      action,
		Payload:     payload,
		CreatedAt:   createdAt,
	}
}

func (m *historyModel) toDomain() *Entry {
	if m == nil {
		return nil
	}

	payload := make(map[string]*string, len(m.Payload))
	maps.Copy(payload, m.Payload)

	return &Entry{
		ID: m.ID,
		EntryBase: EntryBase{
			SubjectType: m.SubjectType,
			SubjectID:   m.SubjectID,
			ActorID:     m.ActorID,
			Action:      m.Action,
			Payload:     payload,
		},
		CreatedAt: m.CreatedAt,
	}
}
