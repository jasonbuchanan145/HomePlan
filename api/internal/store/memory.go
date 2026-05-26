package store

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"homeplan/api/internal/house"
	"homeplan/api/internal/identity"
)

type MemoryStore struct {
	mu         sync.RWMutex
	sessions   map[string]time.Time
	states     map[string]json.RawMessage
	devState   json.RawMessage
	users      map[string]identity.AuthUser
	identity   map[string]string
	auth       map[string]memoryAuthSession
	userStates map[string]json.RawMessage
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions:   map[string]time.Time{},
		states:     map[string]json.RawMessage{},
		users:      map[string]identity.AuthUser{},
		identity:   map[string]string{},
		auth:       map[string]memoryAuthSession{},
		userStates: map[string]json.RawMessage{},
	}
}

type memoryAuthSession struct {
	userID    string
	expiresAt time.Time
	revoked   bool
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

func (s *MemoryStore) LoadUserHouse(_ context.Context, userID string) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.userStates[userID]
	if !ok {
		return nil, house.ErrNotFound
	}
	return append(json.RawMessage(nil), state...), nil
}

func (s *MemoryStore) SaveUserHouse(_ context.Context, userID string, state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userStates[userID] = append(json.RawMessage(nil), state...)
	return nil
}

func (s *MemoryStore) DeleteUserHouse(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.userStates, userID)
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

func (s *MemoryStore) UpsertAuthIdentity(_ context.Context, authIdentity identity.AuthIdentity) (identity.AuthUser, identity.AppEntitlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := authIdentity.Provider + ":" + authIdentity.ProviderSubject
	userID := s.identity[key]
	if userID == "" {
		userID = "user-" + authIdentity.ProviderSubject
		s.identity[key] = userID
	}
	user := identity.AuthUser{
		ID:          userID,
		Email:       strings.ToLower(authIdentity.Email),
		DisplayName: authIdentity.DisplayName,
		AvatarURL:   authIdentity.AvatarURL,
	}
	s.users[userID] = user
	return user, identity.AppEntitlement{CanAccess: true, CanUseAI: false}, nil
}

func (s *MemoryStore) CreateUserSession(_ context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth[tokenHash] = memoryAuthSession{userID: userID, expiresAt: expiresAt}
	return nil
}

func (s *MemoryStore) LoadUserSession(_ context.Context, tokenHash string) (identity.SignedInSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.auth[tokenHash]
	if !ok || session.revoked || time.Now().After(session.expiresAt) {
		return identity.SignedInSession{}, house.ErrNotFound
	}
	user, ok := s.users[session.userID]
	if !ok {
		return identity.SignedInSession{}, house.ErrNotFound
	}
	return identity.SignedInSession{
		User:        user,
		Entitlement: identity.AppEntitlement{CanAccess: true, CanUseAI: false},
	}, nil
}

func (s *MemoryStore) RevokeUserSession(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.auth[tokenHash]
	session.revoked = true
	s.auth[tokenHash] = session
	return nil
}
