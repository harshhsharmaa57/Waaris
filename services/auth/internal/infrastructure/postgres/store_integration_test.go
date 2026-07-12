//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/auth/internal/domain"
	"github.com/waaris/waaris/services/auth/internal/infrastructure/postgres"
)

func TestStoreRefreshRotation(t *testing.T) {
	databaseURL := os.Getenv("AUTH_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AUTH_INTEGRATION_DATABASE_URL is required for PostgreSQL integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `DROP SCHEMA IF EXISTS waaris CASCADE; CREATE SCHEMA waaris; CREATE TABLE waaris.users (id UUID PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, display_name TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()); CREATE TABLE waaris.refresh_tokens (id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES waaris.users(id) ON DELETE CASCADE, token_hash CHAR(64) NOT NULL UNIQUE, expires_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(pool)
	user, err := store.CreateUser(ctx, "person@example.com", "hash", "Person")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err = store.CreateRefreshToken(ctx, domain.RefreshToken{ID: uuid.New(), UserID: user.ID, Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	rotated, err := store.RotateRefreshToken(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", uuid.New(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", now.Add(time.Hour), now)
	if err != nil || rotated.ID != user.ID {
		t.Fatalf("rotation failed: %#v, %v", rotated, err)
	}
	if _, err = store.RotateRefreshToken(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", uuid.New(), "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", now.Add(time.Hour), now); err == nil {
		t.Fatal("rotated token remained valid")
	}
}
