package domain

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("digital will already exists")
)

type WillStatus string

const (
	StatusDraft     WillStatus = "draft"
	StatusPublished WillStatus = "published"
)

type ReleaseCategory string

const (
	CategoryFinancial          ReleaseCategory = "financial"
	CategoryPrivate            ReleaseCategory = "private"
	CategoryCommunityShareable ReleaseCategory = "community_shareable"
)

const ConsentTypeTerms = "digital_will_terms"

type UpsertWillInput struct {
	Status                WillStatus
	DormancyPeriodDays    int
	GracePeriodDays       int
	PolicyVersionAccepted string
	ReleaseCategories     []ReleaseCategory
}

type DigitalWill struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	Status                WillStatus
	Version               int
	DormancyPeriodDays    int
	GracePeriodDays       int
	PolicyVersionAccepted string
	ConsentAcceptedAt     time.Time
	ReleaseCategories     []ReleaseCategory
	CreatedAt             time.Time
	UpdatedAt             time.Time
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

func CloneCategories(values []ReleaseCategory) []ReleaseCategory {
	return slices.Clone(values)
}
