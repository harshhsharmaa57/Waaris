package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

type Service struct {
	store      Store
	tokens     *TokenManager
	refreshTTL time.Duration
	now        func() time.Time
}

type Session struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func NewService(store Store, tokens *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{store: store, tokens: tokens, refreshTTL: refreshTTL, now: time.Now}
}

func (s *Service) Register(ctx context.Context, email, password, displayName string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := hashPassword(password)
	if err != nil {
		return Session{}, err
	}
	user, err := s.store.CreateUser(ctx, email, hash, strings.TrimSpace(displayName))
	if err != nil {
		return Session{}, err
	}
	session, err := s.newSession(ctx, user)
	if err != nil {
		return Session{}, err
	}
	if err = s.store.AppendAuditEvent(ctx, user.ID, "user", user.Email, "registration", s.now()); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := s.store.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || comparePassword(user.PasswordHash, password) != nil {
		return Session{}, domain.ErrInvalidCredentials
	}
	session, err := s.newSession(ctx, user.User)
	if err != nil {
		return Session{}, err
	}
	if err = s.store.AppendAuditEvent(ctx, user.ID, "user", user.Email, "login", s.now()); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	now := s.now()
	placeholder, next, err := NewRefreshToken(uuid.Nil, s.refreshTTL, now)
	if err != nil {
		return Session{}, err
	}
	user, err := s.store.RotateRefreshToken(ctx, HashRefreshToken(refreshToken), next.ID, next.Hash, next.ExpiresAt, now)
	if err != nil {
		return Session{}, err
	}
	access, expiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return Session{}, err
	}
	return Session{User: user, AccessToken: access, RefreshToken: placeholder, ExpiresAt: expiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.RevokeRefreshToken(ctx, HashRefreshToken(refreshToken), s.now())
}

func (s *Service) Profile(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	return s.store.UserByID(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName string) (domain.User, error) {
	return s.store.UpdateProfile(ctx, userID, strings.TrimSpace(displayName))
}

func (s *Service) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	return s.store.DeleteUser(ctx, userID)
}

func (s *Service) ParseAccessToken(raw string) (Principal, error) {
	return s.tokens.ParseAccessToken(raw)
}

func (s *Service) newSession(ctx context.Context, user domain.User) (Session, error) {
	now := s.now()
	refresh, refreshRecord, err := NewRefreshToken(user.ID, s.refreshTTL, now)
	if err != nil {
		return Session{}, err
	}
	if err := s.store.CreateRefreshToken(ctx, refreshRecord); err != nil {
		return Session{}, err
	}
	access, expiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return Session{}, err
	}
	return Session{User: user, AccessToken: access, RefreshToken: refresh, ExpiresAt: expiresAt}, nil
}
