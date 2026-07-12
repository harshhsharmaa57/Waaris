package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/waaris/waaris/services/auth/internal/application"
	"github.com/waaris/waaris/services/auth/internal/infrastructure/memory"
	"github.com/waaris/waaris/services/auth/internal/transport/httpapi"
)

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	tokens, err := application.NewTokenManager("12345678901234567890123456789012", "waaris-auth", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.NewHandler(application.NewService(memory.New(), tokens, time.Hour)).Router()
}

func request(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestAuthenticationHTTPFlow(t *testing.T) {
	router := newRouter(t)
	registered := request(t, router, http.MethodPost, "/v1/auth/register", map[string]string{"email": "person@example.com", "password": "correct-horse-battery", "displayName": "Person"}, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status: %d", registered.Code)
	}
	var session struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(registered.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	profile := request(t, router, http.MethodGet, "/v1/users/me", map[string]string{}, session.AccessToken)
	if profile.Code != http.StatusOK {
		t.Fatalf("profile status: %d", profile.Code)
	}
	updated := request(t, router, http.MethodPatch, "/v1/users/me", map[string]string{"displayName": "Updated"}, session.AccessToken)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status: %d", updated.Code)
	}
	refreshed := request(t, router, http.MethodPost, "/v1/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "")
	if refreshed.Code != http.StatusOK {
		t.Fatalf("refresh status: %d", refreshed.Code)
	}
	if request(t, router, http.MethodPost, "/v1/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "").Code != http.StatusUnauthorized {
		t.Fatal("rotated token remained valid")
	}
}

func TestAuthenticationValidationAndMiddleware(t *testing.T) {
	router := newRouter(t)
	invalid := request(t, router, http.MethodPost, "/v1/auth/register", map[string]string{"email": "bad", "password": "short"}, "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("validation status: %d", invalid.Code)
	}
	unauthorized := request(t, router, http.MethodGet, "/v1/users/me", map[string]string{}, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("middleware status: %d", unauthorized.Code)
	}
}
