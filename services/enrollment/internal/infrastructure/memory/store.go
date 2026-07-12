package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type storedWill struct {
	current domain.DigitalWill
	history []domain.WillVersion
	deleted bool
}

type Store struct {
	mu    sync.Mutex
	wills map[uuid.UUID]storedWill
}

func New() *Store {
	return &Store{wills: map[uuid.UUID]storedWill{}}
}

func (s *Store) CreateWill(_ context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.wills[userID]; ok && !existing.deleted {
		return domain.DigitalWill{}, domain.ErrAlreadyExists
	}

	willID := uuid.New()
	versionID := uuid.New()
	current := domain.DigitalWill{
		ID:                    willID,
		UserID:                userID,
		Status:                input.Status,
		Version:               1,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	history := []domain.WillVersion{{
		ID:                    versionID,
		WillID:                willID,
		UserID:                userID,
		Version:               1,
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
	}}
	s.wills[userID] = storedWill{current: current, history: history}
	return current, nil
}

func (s *Store) WillByUser(_ context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.DigitalWill{}, domain.ErrNotFound
	}
	result := value.current
	result.ReleaseCategories = domain.CloneCategories(result.ReleaseCategories)
	return result, nil
}

func (s *Store) UpdateWill(_ context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.DigitalWill{}, domain.ErrNotFound
	}

	value.current.Version++
	value.current.Status = input.Status
	value.current.DormancyPeriodDays = input.DormancyPeriodDays
	value.current.GracePeriodDays = input.GracePeriodDays
	value.current.PolicyVersionAccepted = input.PolicyVersionAccepted
	value.current.ConsentAcceptedAt = now
	value.current.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	value.current.UpdatedAt = now
	value.history = append(value.history, domain.WillVersion{
		ID:                    uuid.New(),
		WillID:                value.current.ID,
		UserID:                userID,
		Version:               value.current.Version,
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
	})
	s.wills[userID] = value

	result := value.current
	result.ReleaseCategories = domain.CloneCategories(result.ReleaseCategories)
	return result, nil
}

func (s *Store) DeleteWill(_ context.Context, userID uuid.UUID, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.ErrNotFound
	}
	value.deleted = true
	s.wills[userID] = value
	return nil
}

func (s *Store) WillHistory(_ context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return nil, domain.ErrNotFound
	}

	result := make([]domain.WillVersion, 0, len(value.history))
	for index := len(value.history) - 1; index >= 0; index-- {
		item := value.history[index]
		item.ReleaseCategories = domain.CloneCategories(item.ReleaseCategories)
		result = append(result, item)
	}
	return result, nil
}
