package store

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"homeplan/api/internal/house"
)

type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
	states   map[string]json.RawMessage
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: map[string]time.Time{},
		states:   map[string]json.RawMessage{},
	}
}

func (s *MemoryStore) EnsureAnonymousSession(_ context.Context, token string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = expiresAt
	return nil
}

func (s *MemoryStore) LoadCurrentHouse(_ context.Context, sessionToken string) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[sessionToken]
	if !ok {
		return nil, house.ErrNotFound
	}
	return append(json.RawMessage(nil), state...), nil
}

func (s *MemoryStore) SaveCurrentHouse(_ context.Context, sessionToken string, state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[sessionToken] = append(json.RawMessage(nil), state...)
	return nil
}
