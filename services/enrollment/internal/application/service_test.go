package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/memory"
)

type noopNotifier struct{}

func (noopNotifier) Send(context.Context, domain.Notification) error { return nil }

func newService() *application.Service {
	store := memory.New()
	service := application.NewService(store, store, noopNotifier{})
	return service
}

func TestWillLifecycleAndHistory(t *testing.T) {
	service := newService()
	userID := uuid.New()

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
	if created.Version != 1 || created.LifecycleState != domain.LifecycleActive {
		t.Fatalf("unexpected create result: %#v", created)
	}

	if _, err = service.CreateTrustee(context.Background(), userID, domain.TrusteeInput{Name: "Trustee One", Email: "trustee@example.com", Relationship: "Sibling"}); err != nil {
		t.Fatal(err)
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

func TestWillValidationTrusteesAndHeartbeat(t *testing.T) {
	store := memory.New()
	service := application.NewService(store, store, noopNotifier{})

	userID := uuid.New()
	if _, err := service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); !errors.Is(err, domain.ErrPublishedRequiresTrustee) {
		t.Fatalf("got %v, want published_requires_trustee", err)
	}

	if _, err := service.CreateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTrustee(context.Background(), userID, domain.TrusteeInput{Name: "Trustee One", Email: "trustee@example.com", Relationship: "Friend"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTrustee(context.Background(), userID, domain.TrusteeInput{Name: "Trustee One", Email: "trustee@example.com", Relationship: "Friend"}); !errors.Is(err, domain.ErrDuplicateTrustee) {
		t.Fatalf("got %v, want duplicate trustee", err)
	}
	if _, err := service.UpdateWill(context.Background(), userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	status, err := service.SubmitHeartbeat(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastHeartbeatAt == nil {
		t.Fatal("expected heartbeat timestamp")
	}
}

func TestLifecycleStateMachineAndConfigurationLock(t *testing.T) {
	store := memory.New()
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	service := application.NewServiceWithClock(store, store, noopNotifier{}, func() time.Time { return now })
	ownerID := uuid.New()
	trusteeID := uuid.New()

	if _, err := service.CreateWill(context.Background(), ownerID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTrustee(context.Background(), ownerID, domain.TrusteeInput{Name: "Trustee", Email: "trustee@example.com", Relationship: "Sibling"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateWill(context.Background(), ownerID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(48 * time.Hour)
	if err := service.ProcessLifecycleTick(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err := service.HeartbeatStatus(context.Background(), ownerID)
	if err != nil || status.LifecycleState != domain.LifecyclePendingVerification {
		t.Fatalf("expected pending verification, got %#v, %v", status, err)
	}
	if _, err = service.UpdateWill(context.Background(), ownerID, domain.UpsertWillInput{}); !errors.Is(err, domain.ErrWillNotEditable) {
		t.Fatalf("got %v, want locked will", err)
	}

	pending, err := service.PendingVerifications(context.Background(), "trustee@example.com")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending requests: %#v, %v", pending, err)
	}
	if err = service.ApproveVerification(context.Background(), "trustee@example.com", trusteeID, pending[0].ID); err != nil {
		t.Fatal(err)
	}
	status, err = service.HeartbeatStatus(context.Background(), ownerID)
	if err != nil || status.LifecycleState != domain.LifecycleGracePeriod {
		t.Fatalf("expected grace period, got %#v, %v", status, err)
	}

	now = now.Add(48 * time.Hour)
	if err = service.ProcessLifecycleTick(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err = service.HeartbeatStatus(context.Background(), ownerID)
	if err != nil || status.LifecycleState != domain.LifecycleReadyForExecution {
		t.Fatalf("expected ready for execution, got %#v, %v", status, err)
	}
	if _, err = service.SubmitHeartbeat(context.Background(), ownerID); err != nil {
		t.Fatal(err)
	}
	status, err = service.HeartbeatStatus(context.Background(), ownerID)
	if err != nil || status.LifecycleState != domain.LifecycleActive {
		t.Fatalf("expected active recovery, got %#v, %v", status, err)
	}
}

func TestTrusteeCannotBeWillOwner(t *testing.T) {
	service := newService()
	ownerID := uuid.New()
	if _, err := service.CreateWill(context.Background(), ownerID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := service.CreateTrustee(context.Background(), ownerID, domain.TrusteeInput{Name: "Owner", Email: ownerID.String() + "@example.com", Relationship: "Self"})
	if !errors.Is(err, domain.ErrSelfTrustee) {
		t.Fatalf("got %v, want self trustee error", err)
	}
}
