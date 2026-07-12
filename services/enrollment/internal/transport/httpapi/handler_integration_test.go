package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/memory"
	"github.com/waaris/waaris/services/enrollment/internal/transport/httpapi"
)

const testSecret = "12345678901234567890123456789012"

type noopNotifier struct{}

func (noopNotifier) Send(context.Context, domain.Notification) error { return nil }

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	verifier, err := application.NewTokenVerifier(testSecret, "waaris-auth")
	if err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	return httpapi.NewHandler(application.NewService(store, store, noopNotifier{}), verifier).Router()
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

func TestEnrollmentHTTPFlow(t *testing.T) {
	router := newRouter(t)
	token := accessToken(t, uuid.New(), "person@example.com")

	created := request(t, router, http.MethodPost, "/api/v1/will", map[string]any{
		"state":                 "draft",
		"dormancyPeriodDays":    180,
		"gracePeriodDays":       30,
		"policyVersionAccepted": "2026-07",
		"releaseCategories":     []string{"financial", "private"},
	}, token)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status: %d", created.Code)
	}

	trustee := request(t, router, http.MethodPost, "/api/v1/trustees", map[string]any{
		"name":         "Trustee One",
		"email":        "trustee@example.com",
		"relationship": "Sibling",
	}, token)
	if trustee.Code != http.StatusCreated {
		t.Fatalf("trustee status: %d", trustee.Code)
	}
	var trusteeBody struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(trustee.Body).Decode(&trusteeBody); err != nil {
		t.Fatal(err)
	}
	otherUser := accessToken(t, uuid.New(), "other@example.com")
	if response := request(t, router, http.MethodDelete, "/api/v1/trustees/"+trusteeBody.ID, map[string]string{}, otherUser); response.Code != http.StatusNotFound {
		t.Fatalf("cross-user trustee delete status: %d", response.Code)
	}

	updated := request(t, router, http.MethodPut, "/api/v1/will", map[string]any{
		"state":                 "published",
		"dormancyPeriodDays":    365,
		"gracePeriodDays":       45,
		"policyVersionAccepted": "2026-08",
		"releaseCategories":     []string{"community_shareable"},
	}, token)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status: %d", updated.Code)
	}

	heartbeat := request(t, router, http.MethodPost, "/api/v1/heartbeats", map[string]string{}, token)
	if heartbeat.Code != http.StatusCreated {
		t.Fatalf("heartbeat status: %d", heartbeat.Code)
	}

	history := request(t, router, http.MethodGet, "/api/v1/will/history", map[string]string{}, token)
	if history.Code != http.StatusOK {
		t.Fatalf("history status: %d", history.Code)
	}
}

func TestEnrollmentHTTPValidationAndContract(t *testing.T) {
	router := newRouter(t)
	token := accessToken(t, uuid.New(), "person@example.com")

	unauthorized := request(t, router, http.MethodGet, "/api/v1/will", map[string]string{}, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status: %d", unauthorized.Code)
	}

	invalid := request(t, router, http.MethodPost, "/api/v1/trustees", map[string]any{
		"name":         "",
		"email":        "bad",
		"relationship": "Sibling",
	}, token)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("validation status: %d", invalid.Code)
	}

	var errorBody map[string]any
	if err := json.NewDecoder(invalid.Body).Decode(&errorBody); err != nil {
		t.Fatal(err)
	}
	if errorBody["code"] == "" || errorBody["message"] == "" || errorBody["correlationId"] == "" {
		t.Fatalf("unexpected error contract: %#v", errorBody)
	}
	if invalid.Header().Get("X-Content-Type-Options") != "nosniff" || invalid.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("missing security headers: %#v", invalid.Header())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/will", bytes.NewBufferString(`{} {}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Correlation-Id", "invalid value")
	trailing := httptest.NewRecorder()
	router.ServeHTTP(trailing, req)
	if trailing.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status: %d", trailing.Code)
	}
	if trailing.Header().Get("X-Correlation-Id") == "invalid value" {
		t.Fatal("unsafe correlation ID was reflected")
	}
}

func accessToken(t *testing.T, userID uuid.UUID, email string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"iss":   "waaris-auth",
		"iat":   1,
		"exp":   4102444800,
	})
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
