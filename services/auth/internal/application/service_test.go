package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/waaris/waaris/services/auth/internal/application"
	"github.com/waaris/waaris/services/auth/internal/domain"
	"github.com/waaris/waaris/services/auth/internal/infrastructure/memory"
)

func newService(t *testing.T) *application.Service {
	t.Helper()
	tokens, err := application.NewTokenManager("12345678901234567890123456789012", "test", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return application.NewService(memory.New(), tokens, time.Hour)
}

func TestRegisterLoginAndRefreshRotation(t *testing.T) {
	service := newService(t)
	registered, err := service.Register(context.Background(), "person@example.com", "correct-horse-battery", "Person")
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.Email != "person@example.com" || registered.AccessToken == "" || registered.RefreshToken == "" {
		t.Fatalf("unexpected registration session: %#v", registered)
	}
	if _, err = service.Login(context.Background(), "person@example.com", "wrong-password"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want invalid credentials", err)
	}
	loggedIn, err := service.Login(context.Background(), "PERSON@example.com", "correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := service.Refresh(context.Background(), loggedIn.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.RefreshToken == loggedIn.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err = service.Refresh(context.Background(), loggedIn.RefreshToken); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("got %v, want invalid refresh token", err)
	}
}

func TestProfileLifecycle(t *testing.T) {
	service := newService(t)
	session, err := service.Register(context.Background(), "person@example.com", "correct-horse-battery", "Person")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateProfile(context.Background(), session.User.ID, "Updated Person")
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "Updated Person" {
		t.Fatalf("got %q", updated.DisplayName)
	}
	if err = service.DeleteProfile(context.Background(), session.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Profile(context.Background(), session.User.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want not found", err)
	}
}
