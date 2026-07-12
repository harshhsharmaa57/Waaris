package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

type Store interface {
	CreateUser(context.Context, string, string, string) (domain.User, error)
	UserByEmail(context.Context, string) (domain.UserWithPassword, error)
	UserByID(context.Context, uuid.UUID) (domain.User, error)
	UpdateProfile(context.Context, uuid.UUID, string) (domain.User, error)
	DeleteUser(context.Context, uuid.UUID) error
	CreateRefreshToken(context.Context, domain.RefreshToken) error
	RotateRefreshToken(context.Context, string, uuid.UUID, string, time.Time, time.Time) (domain.User, error)
	RevokeRefreshToken(context.Context, string, time.Time) error
}
