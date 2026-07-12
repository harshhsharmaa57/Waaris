package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/memory"
)

func newService() *application.Service {
	service := application.NewService(memory.New())
	return service
}

func TestWillLifecycleAndHistory(t *testing.T) {
	service := newService()
	userID := mustUUID(t)

	created, err := service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    180,
		GracePeriodDays:       30,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryPrivate, domain.CategoryFinancial},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Version != 1 {
		t.Fatalf("got version %d", created.Version)
	}

	updated, err := service.UpdateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    365,
		GracePeriodDays:       45,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial, domain.CategoryCommunityShareable},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.Status != domain.StatusPublished {
		t.Fatalf("unexpected updated will: %#v", updated)
	}

	history, err := service.WillHistory(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Version != 2 || history[1].Version != 1 {
		t.Fatalf("unexpected history: %#v", history)
	}

	if err = service.DeleteWill(context.Background(), userID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Will(context.Background(), userID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want not found", err)
	}
}

func TestWillValidationAndSingleActiveWill(t *testing.T) {
	service := newService()
	userID := mustUUID(t)

	if _, err := service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    0,
		GracePeriodDays:       30,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err == nil {
		t.Fatal("expected validation error")
	}

	_, err := service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    180,
		GracePeriodDays:       30,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    180,
		GracePeriodDays:       30,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("got %v, want already exists", err)
	}
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.New()
}
