package webauthn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-core-fx/cachefx/cache"
	"github.com/go-webauthn/webauthn/webauthn"
)

const sessionTTL = 5 * time.Minute

type sessionEntry struct {
	*webauthn.SessionData
}

func (e *sessionEntry) Marshal() ([]byte, error) {
	b, err := json.Marshal(e.SessionData)
	if err != nil {
		return nil, fmt.Errorf("marshal session data: %w", err)
	}
	return b, nil
}

func (e *sessionEntry) Unmarshal(data []byte) error {
	e.SessionData = &webauthn.SessionData{} //nolint:exhaustruct // populated by Unmarshal
	if err := json.Unmarshal(data, e.SessionData); err != nil {
		return fmt.Errorf("unmarshal session data: %w", err)
	}
	return nil
}

type sessions struct {
	storage *cache.Typed[*sessionEntry]
}

func newSessions(storage cache.Cache) *sessions {
	return &sessions{storage: cache.NewTyped[*sessionEntry](storage)}
}

func (s *sessions) Store(ctx context.Context, data *webauthn.SessionData) error {
	if err := s.storage.Set(
		ctx,
		data.Challenge,
		&sessionEntry{SessionData: data},
		cache.WithTTL(sessionTTL),
	); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}
	return nil
}

func (s *sessions) Consume(ctx context.Context, challenge string) (*webauthn.SessionData, error) {
	entry, err := s.storage.Get(ctx, challenge, cache.AndDelete())
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	return entry.SessionData, nil
}
