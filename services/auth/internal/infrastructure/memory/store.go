package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

type Store struct {
	mu      sync.Mutex
	users   map[uuid.UUID]domain.UserWithPassword
	byEmail map[string]uuid.UUID
	refresh map[string]domain.RefreshToken
}

func New() *Store {
	return &Store{users: map[uuid.UUID]domain.UserWithPassword{}, byEmail: map[string]uuid.UUID{}, refresh: map[string]domain.RefreshToken{}}
}

func (s *Store) CreateUser(_ context.Context, email, passwordHash, displayName string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[email]; exists {
		return domain.User{}, domain.ErrEmailTaken
	}
	now := time.Now().UTC()
	user := domain.User{ID: uuid.New(), Email: email, DisplayName: displayName, CreatedAt: now, UpdatedAt: now}
	s.users[user.ID] = domain.UserWithPassword{User: user, PasswordHash: passwordHash}
	s.byEmail[email] = user.ID
	return user, nil
}

func (s *Store) UserByEmail(_ context.Context, email string) (domain.UserWithPassword, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byEmail[email]
	if !ok {
		return domain.UserWithPassword{}, domain.ErrNotFound
	}
	return s.users[id], nil
}

func (s *Store) UserByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user.User, nil
}

func (s *Store) UpdateProfile(_ context.Context, id uuid.UUID, displayName string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	user.DisplayName = displayName
	user.UpdatedAt = time.Now().UTC()
	s.users[id] = user
	return user.User, nil
}

func (s *Store) DeleteUser(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(s.users, id)
	delete(s.byEmail, user.Email)
	for hash, token := range s.refresh {
		if token.UserID == id {
			delete(s.refresh, hash)
		}
	}
	return nil
}

func (s *Store) CreateRefreshToken(_ context.Context, token domain.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh[token.Hash] = token
	return nil
}

func (s *Store) RotateRefreshToken(_ context.Context, hash string, nextID uuid.UUID, nextHash string, expiresAt, now time.Time) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.refresh[hash]
	if !ok || now.After(current.ExpiresAt) {
		return domain.User{}, domain.ErrInvalidRefreshToken
	}
	delete(s.refresh, hash)
	user, ok := s.users[current.UserID]
	if !ok {
		return domain.User{}, domain.ErrInvalidRefreshToken
	}
	s.refresh[nextHash] = domain.RefreshToken{ID: nextID, UserID: user.ID, Hash: nextHash, CreatedAt: now.UTC(), ExpiresAt: expiresAt}
	return user.User, nil
}

func (s *Store) RevokeRefreshToken(_ context.Context, hash string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refresh, hash)
	return nil
}
