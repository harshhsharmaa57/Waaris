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
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/postgres"
)

func TestMigrationCreatesMVPTables(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()

	for _, name := range []string{
		"waaris.digital_wills",
		"waaris.will_versions",
		"waaris.consent_records",
		"waaris.trustees",
		"waaris.heartbeats",
		"waaris.verification_requests",
		"waaris.verification_responses",
		"waaris.notifications",
		"waaris.audit_events",
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

func TestStoreMVPWorkflow(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "owner@example.com")
	trusteeUserID := seedUser(t, pool, "trustee@example.com")

	repository := postgres.New(pool)
	service := application.NewService(repository, repository, noopNotifier{})

	if _, err := service.CreateWill(ctx, userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTrustee(ctx, userID, domain.TrusteeInput{Name: "Trustee", Email: "trustee@example.com", Relationship: "Sibling"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateWill(ctx, userID, domain.UpsertWillInput{
		Status:                domain.StatusPublished,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-08",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE waaris.digital_wills SET updated_at = NOW() - INTERVAL '2 day' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProcessLifecycleTick(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := service.PendingVerifications(ctx, "trustee@example.com")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending verifications: %#v, %v", pending, err)
	}
	if err = service.ApproveVerification(ctx, "trustee@example.com", trusteeUserID, pending[0].ID); err != nil {
		t.Fatal(err)
	}
	status, err := service.HeartbeatStatus(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LifecycleState != domain.LifecycleGracePeriod {
		t.Fatalf("expected grace period, got %s", status.LifecycleState)
	}
}

func TestDuplicateTrusteeRollsBack(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "owner@example.com")
	repository := postgres.New(pool)
	service := application.NewService(repository, repository, noopNotifier{})
	if _, err := service.CreateWill(ctx, userID, domain.UpsertWillInput{
		Status:                domain.StatusDraft,
		DormancyPeriodDays:    1,
		GracePeriodDays:       1,
		PolicyVersionAccepted: "2026-07",
		ReleaseCategories:     []domain.ReleaseCategory{domain.CategoryFinancial},
	}); err != nil {
		t.Fatal(err)
	}
	input := domain.TrusteeInput{Name: "Trustee", Email: "trustee@example.com", Relationship: "Sibling"}
	if _, err := service.CreateTrustee(ctx, userID, input); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTrustee(ctx, userID, input); err == nil {
		t.Fatal("duplicate trustee was accepted")
	}
	trustees, err := service.Trustees(ctx, userID)
	if err != nil || len(trustees) != 1 {
		t.Fatalf("duplicate request changed data: %#v, %v", trustees, err)
	}
}

type noopNotifier struct{}

func (noopNotifier) Send(context.Context, domain.Notification) error { return nil }

func seedUser(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO waaris.users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'hash', 'Person')
	`, userID, email); err != nil {
		t.Fatal(err)
	}
	return userID
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
		"000004_mvp_workflow.up.sql",
		"000005_mvp_hardening.up.sql",
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
