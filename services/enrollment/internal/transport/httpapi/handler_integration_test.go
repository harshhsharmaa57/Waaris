package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/infrastructure/memory"
	"github.com/waaris/waaris/services/enrollment/internal/transport/httpapi"
)

const testSecret = "12345678901234567890123456789012"

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	verifier, err := application.NewTokenVerifier(testSecret, "waaris-auth")
	if err != nil {
		t.Fatal(err)
	}
	return httpapi.NewHandler(application.NewService(memory.New()), verifier).Router()
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

func TestWillHTTPFlow(t *testing.T) {
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

	current := request(t, router, http.MethodGet, "/api/v1/will", map[string]string{}, token)
	if current.Code != http.StatusOK {
		t.Fatalf("get status: %d", current.Code)
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

	history := request(t, router, http.MethodGet, "/api/v1/will/history", map[string]string{}, token)
	if history.Code != http.StatusOK {
		t.Fatalf("history status: %d", history.Code)
	}
	var payload struct {
		History []struct {
			Version int `json:"version"`
		} `json:"history"`
	}
	if err := json.NewDecoder(history.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.History) != 2 || payload.History[0].Version != 2 {
		t.Fatalf("unexpected history payload: %#v", payload)
	}
}

func TestWillHTTPValidationAndContract(t *testing.T) {
	router := newRouter(t)
	token := accessToken(t, uuid.New(), "person@example.com")

	unauthorized := request(t, router, http.MethodGet, "/api/v1/will", map[string]string{}, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status: %d", unauthorized.Code)
	}

	invalid := request(t, router, http.MethodPost, "/api/v1/will", map[string]any{
		"state":                 "draft",
		"dormancyPeriodDays":    0,
		"gracePeriodDays":       30,
		"policyVersionAccepted": "2026-07",
		"releaseCategories":     []string{"financial"},
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
