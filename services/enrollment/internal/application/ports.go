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
}
