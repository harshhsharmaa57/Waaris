package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type Store interface {
	CreateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error)
	WillByUser(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error)
	UpdateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error)
	DeleteWill(ctx context.Context, userID uuid.UUID, now time.Time) error
	WillHistory(ctx context.Context, userID uuid.UUID) ([]domain.WillVersion, error)
	TrusteeCount(ctx context.Context, userID uuid.UUID) (int, error)
	CreateTrustee(ctx context.Context, userID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error)
	TrusteesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Trustee, error)
	UpdateTrustee(ctx context.Context, userID uuid.UUID, trusteeID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error)
	DeleteTrustee(ctx context.Context, userID uuid.UUID, trusteeID uuid.UUID, now time.Time) error
	SubmitHeartbeat(ctx context.Context, userID uuid.UUID, source string, now time.Time) (domain.HeartbeatStatus, bool, error)
	HeartbeatStatus(ctx context.Context, userID uuid.UUID) (domain.HeartbeatStatus, error)
	HeartbeatHistory(ctx context.Context, userID uuid.UUID) ([]domain.Heartbeat, error)
	TransitionOverdueWills(ctx context.Context, now time.Time) ([]LifecycleEvent, error)
	PendingVerifications(ctx context.Context, trusteeEmail string) ([]domain.VerificationRequest, error)
	RecordVerificationDecision(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID, decision domain.VerificationDecision, now time.Time) (VerificationDecisionResult, error)
	NotificationHistory(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error)
	AuditHistory(ctx context.Context, userID uuid.UUID) ([]domain.AuditEvent, error)
}

type NotificationQueue interface {
	Enqueue(ctx context.Context, notifications []domain.Notification) error
	Pending(ctx context.Context, limit int) ([]domain.Notification, error)
	MarkSent(ctx context.Context, notificationID uuid.UUID, sentAt time.Time) error
	MarkFailed(ctx context.Context, notificationID uuid.UUID, message string, failedAt time.Time) error
}

type Notifier interface {
	Send(ctx context.Context, notification domain.Notification) error
}

type LifecycleEvent struct {
	UserID              uuid.UUID
	WillID              uuid.UUID
	OwnerEmail          string
	OwnerDisplayName    string
	ThresholdRequired   int
	Trustees            []domain.Trustee
	TransitionedAt      time.Time
	PreviousState       domain.LifecycleState
	CurrentState        domain.LifecycleState
	GracePeriodDays     int
	VerificationRequest *uuid.UUID
}

type VerificationDecisionResult struct {
	UserID            uuid.UUID
	WillID            uuid.UUID
	OwnerEmail        string
	OwnerDisplayName  string
	Trustees          []domain.Trustee
	TransitionedTo    *domain.LifecycleState
	GracePeriodDays   int
	RequestID         uuid.UUID
	Decision          domain.VerificationDecision
	ActorTrustee      domain.Trustee
	Approvals         int
	ThresholdRequired int
}
