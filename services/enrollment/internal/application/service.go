package application

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

var allowedCategories = map[domain.ReleaseCategory]struct{}{
	domain.CategoryFinancial:          {},
	domain.CategoryPrivate:            {},
	domain.CategoryCommunityShareable: {},
}

type ValidationError struct{ message string }

func (e ValidationError) Error() string { return e.message }

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) CreateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput) (domain.DigitalWill, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	return s.store.CreateWill(ctx, userID, normalized, s.now().UTC())
}

func (s *Service) Will(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	return s.store.WillByUser(ctx, userID)
}

func (s *Service) UpdateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput) (domain.DigitalWill, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	return s.store.UpdateWill(ctx, userID, normalized, s.now().UTC())
}

func (s *Service) DeleteWill(ctx context.Context, userID uuid.UUID) error {
	return s.store.DeleteWill(ctx, userID, s.now().UTC())
}

func (s *Service) WillHistory(ctx context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	return s.store.WillHistory(ctx, userID)
}

func normalizeInput(input domain.UpsertWillInput) (domain.UpsertWillInput, error) {
	normalized := domain.UpsertWillInput{
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: strings.TrimSpace(input.PolicyVersionAccepted),
	}

	if input.Status != domain.StatusDraft && input.Status != domain.StatusPublished {
		return domain.UpsertWillInput{}, ValidationError{message: "state must be either draft or published"}
	}
	if input.DormancyPeriodDays < 1 || input.DormancyPeriodDays > 3650 {
		return domain.UpsertWillInput{}, ValidationError{message: "dormancyPeriodDays must be between 1 and 3650"}
	}
	if input.GracePeriodDays < 1 || input.GracePeriodDays > 365 {
		return domain.UpsertWillInput{}, ValidationError{message: "gracePeriodDays must be between 1 and 365"}
	}
	if normalized.PolicyVersionAccepted == "" || len(normalized.PolicyVersionAccepted) > 64 {
		return domain.UpsertWillInput{}, ValidationError{message: "policyVersionAccepted is required and must not exceed 64 characters"}
	}

	categories := map[domain.ReleaseCategory]struct{}{}
	for _, category := range input.ReleaseCategories {
		value := domain.ReleaseCategory(strings.TrimSpace(string(category)))
		if _, ok := allowedCategories[value]; !ok {
			return domain.UpsertWillInput{}, ValidationError{message: "releaseCategories contain an unsupported value"}
		}
		categories[value] = struct{}{}
	}
	if len(categories) == 0 {
		return domain.UpsertWillInput{}, ValidationError{message: "at least one release category is required"}
	}

	normalized.ReleaseCategories = make([]domain.ReleaseCategory, 0, len(categories))
	for category := range categories {
		normalized.ReleaseCategories = append(normalized.ReleaseCategories, category)
	}
	slices.Sort(normalized.ReleaseCategories)

	return normalized, nil
}
