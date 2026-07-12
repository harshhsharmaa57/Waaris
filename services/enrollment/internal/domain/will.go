package domain

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound                 = errors.New("not found")
	ErrAlreadyExists            = errors.New("digital will already exists")
	ErrDuplicateTrustee         = errors.New("trustee already exists")
	ErrSelfTrustee              = errors.New("will owner cannot be a trustee")
	ErrPublishedRequiresTrustee = errors.New("published wills require at least one trustee")
	ErrWillNotEditable          = errors.New("will configuration is locked while verification is in progress")
	ErrVerificationNotPending   = errors.New("verification request is not pending")
	ErrTrusteeNotAssigned       = errors.New("trustee is not assigned to this request")
)

type WillStatus string

const (
	StatusDraft     WillStatus = "draft"
	StatusPublished WillStatus = "published"
)

type LifecycleState string

const (
	LifecycleActive              LifecycleState = "active"
	LifecyclePendingVerification LifecycleState = "pending_verification"
	LifecycleGracePeriod         LifecycleState = "grace_period"
	LifecycleReadyForExecution   LifecycleState = "ready_for_execution"
)

type ReleaseCategory string

const (
	CategoryFinancial          ReleaseCategory = "financial"
	CategoryPrivate            ReleaseCategory = "private"
	CategoryCommunityShareable ReleaseCategory = "community_shareable"
)

type VerificationDecision string

const (
	DecisionApprove VerificationDecision = "approve"
	DecisionReject  VerificationDecision = "reject"
	DecisionAbstain VerificationDecision = "abstain"
)

const ConsentTypeTerms = "digital_will_terms"

type UpsertWillInput struct {
	Status                WillStatus
	DormancyPeriodDays    int
	GracePeriodDays       int
	PolicyVersionAccepted string
	ReleaseCategories     []ReleaseCategory
}

type TrusteeInput struct {
	Name         string
	Email        string
	Relationship string
}

type DigitalWill struct {
	ID                           uuid.UUID
	UserID                       uuid.UUID
	OwnerEmail                   string
	OwnerDisplayName             string
	Status                       WillStatus
	LifecycleState               LifecycleState
	Version                      int
	DormancyPeriodDays           int
	GracePeriodDays              int
	PolicyVersionAccepted        string
	ConsentAcceptedAt            time.Time
	LastHeartbeatAt              *time.Time
	PendingVerificationStartedAt *time.Time
	GracePeriodStartedAt         *time.Time
	ReadyForExecutionAt          *time.Time
	ReleaseCategories            []ReleaseCategory
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

type WillVersion struct {
	ID                    uuid.UUID
	WillID                uuid.UUID
	UserID                uuid.UUID
	Version               int
	Status                WillStatus
	DormancyPeriodDays    int
	GracePeriodDays       int
	PolicyVersionAccepted string
	ConsentAcceptedAt     time.Time
	ReleaseCategories     []ReleaseCategory
	CreatedAt             time.Time
}

type ConsentRecord struct {
	ID            uuid.UUID
	WillID        uuid.UUID
	WillVersionID uuid.UUID
	UserID        uuid.UUID
	PolicyVersion string
	ConsentType   string
	AcceptedAt    time.Time
}

type Trustee struct {
	ID           uuid.UUID
	WillID       uuid.UUID
	UserID       uuid.UUID
	Name         string
	Email        string
	Relationship string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Heartbeat struct {
	ID         uuid.UUID
	WillID     uuid.UUID
	UserID     uuid.UUID
	Source     string
	OccurredAt time.Time
	CreatedAt  time.Time
}

type HeartbeatStatus struct {
	WillID                       uuid.UUID
	LifecycleState               LifecycleState
	LastHeartbeatAt              *time.Time
	PendingVerificationStartedAt *time.Time
	GracePeriodStartedAt         *time.Time
	ReadyForExecutionAt          *time.Time
}

type VerificationRequest struct {
	ID                uuid.UUID
	WillID            uuid.UUID
	UserID            uuid.UUID
	ThresholdRequired int
	Status            string
	CreatedAt         time.Time
	ResolvedAt        *time.Time
	OwnerEmail        string
	OwnerDisplayName  string
	Trustee           Trustee
	LatestDecision    *VerificationDecision
}

type VerificationResponse struct {
	ID          uuid.UUID
	RequestID   uuid.UUID
	WillID      uuid.UUID
	TrusteeID   uuid.UUID
	ActorUserID uuid.UUID
	Decision    VerificationDecision
	RespondedAt time.Time
}

type Notification struct {
	ID             uuid.UUID
	WillID         uuid.UUID
	UserID         uuid.UUID
	TrusteeID      *uuid.UUID
	EventType      string
	Channel        string
	RecipientName  string
	RecipientEmail string
	Subject        string
	Body           string
	Status         string
	QueuedAt       time.Time
	SentAt         *time.Time
	FailureMessage string
}

type AuditEvent struct {
	ID            uuid.UUID
	UserID        *uuid.UUID
	WillID        *uuid.UUID
	ActorType     string
	ActorID       string
	EventType     string
	CorrelationID string
	Details       string
	OccurredAt    time.Time
}

func CloneCategories(values []ReleaseCategory) []ReleaseCategory {
	return slices.Clone(values)
}
