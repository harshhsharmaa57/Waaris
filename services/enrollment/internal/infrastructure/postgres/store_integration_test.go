//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/postgres"
)

func TestMigrationCreatesDigitalWillTables(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()

	for _, name := range []string{
		"waaris.digital_wills",
		"waaris.will_versions",
		"waaris.consent_records",
		"waaris.will_release_preferences",
		"waaris.will_version_release_preferences",
	} {
		var tableName string
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1)::text`, name).Scan(&tableName); err != nil {
			t.Fatal(err)
		}
		if tableName != name {
			t.Fatalf("expected table %s", name)
		}
	}
}

func TestStoreWillLifecycle(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO waaris.users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'hash', 'Person')
	`, userID, "person@example.com"); err != nil {
		t.Fatal(err)
	}

	store := postgres.New(pool)
	created, err := store.CreateWill(ctx, userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    180,
		GracePeriodDays:       30,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial, domain.CategoryPrivate},
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.UpdateWill(ctx, userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    365,
		GracePeriodDays:       45,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryCommunityShareable},
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != created.Version+1 {
		t.Fatalf("unexpected version: %d", updated.Version)
	}

	history, err := store.WillHistory(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Version != 2 || history[1].Version != 1 {
		t.Fatalf("unexpected history: %#v", history)
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("ENROLLMENT_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ENROLLMENT_INTEGRATION_DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = pool.Exec(ctx, `DROP SCHEMA IF EXISTS waaris CASCADE;`); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"000001_foundation.up.sql",
		"000002_authentication.up.sql",
		"000003_digital_will_enrollment.up.sql",
	} {
		if _, err = pool.Exec(ctx, readMigration(t, name)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

func readMigration(t *testing.T, name string) string {
	t.Helper()

	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	path := filepath.Join(filepath.Dir(current), "..", "..", "..", "..", "..", "infra", "migrations", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(content))
}
