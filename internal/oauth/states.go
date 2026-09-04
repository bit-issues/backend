package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-core-fx/cachefx/cache"
)

const (
	stateTTL   = 10 * time.Minute
	stateBytes = 32
)

type stateEntry struct {
	UserID int64 `json:"user_id"`
}

func (e *stateEntry) Marshal() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal oauth state: %w", err)
	}
	return b, nil
}

func (e *stateEntry) Unmarshal(data []byte) error {
	if err := json.Unmarshal(data, e); err != nil {
		return fmt.Errorf("unmarshal oauth state: %w", err)
	}
	return nil
}

type stateStore struct {
	storage *cache.Typed[*stateEntry]
}

func newStateStore(c cache.Cache) *stateStore {
	return &stateStore{storage: cache.NewTyped[*stateEntry](c)}
}

// Save persists a single-use CSRF state bound to the initiating user.
func (s *stateStore) Save(ctx context.Context, state string, userID int64) error {
	if err := s.storage.Set(ctx, state, &stateEntry{UserID: userID}, cache.WithTTL(stateTTL)); err != nil {
		return fmt.Errorf("failed to store oauth state: %w", err)
	}
	return nil
}

// Consume reads and deletes the state, returning the bound user. A missing or
// expired state yields ErrStateNotFound, which rejects forged/replayed flows.
func (s *stateStore) Consume(ctx context.Context, state string) (int64, error) {
	entry, err := s.storage.Get(ctx, state, cache.AndDelete())
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return 0, ErrStateNotFound
		}
		return 0, fmt.Errorf("failed to load oauth state: %w", err)
	}
	return entry.UserID, nil
}

// generateState returns a cryptographically random, unguessable state value.
func generateState() (string, error) {
	b := make([]byte, stateBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
