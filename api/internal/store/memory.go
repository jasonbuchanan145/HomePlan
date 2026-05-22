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
	devState json.RawMessage
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

func (s *MemoryStore) DeleteCurrentHouse(_ context.Context, sessionToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, sessionToken)
	return nil
}

func (s *MemoryStore) LoadDevUserHouse(_ context.Context) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.devState) == 0 {
		return nil, house.ErrNotFound
	}
	return append(json.RawMessage(nil), s.devState...), nil
}

func (s *MemoryStore) SaveDevUserHouse(_ context.Context, state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devState = append(json.RawMessage(nil), state...)
	return nil
}

func (s *MemoryStore) ResetDevUserHouse(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devState = nil
	return nil
}
