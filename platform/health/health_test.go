package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerRespondsOnHealthEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler("test-service").Register(mux)

	for _, path := range []string{"/healthz", "/readyz"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s: got status %d, want %d", path, response.Code, http.StatusOK)
		}
	}
}
