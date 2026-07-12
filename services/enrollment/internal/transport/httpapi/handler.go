package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	service *application.Service
	tokens  *application.TokenVerifier
}

func NewHandler(service *application.Service, tokens *application.TokenVerifier) *Handler {
	return &Handler{service: service, tokens: tokens}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "enrollment"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "enrollment"})
	})
	mux.Handle("POST /api/v1/will", h.requireAuth(http.HandlerFunc(h.createWill)))
	mux.Handle("GET /api/v1/will", h.requireAuth(http.HandlerFunc(h.getWill)))
	mux.Handle("PUT /api/v1/will", h.requireAuth(http.HandlerFunc(h.updateWill)))
	mux.Handle("DELETE /api/v1/will", h.requireAuth(http.HandlerFunc(h.deleteWill)))
	mux.Handle("GET /api/v1/will/history", h.requireAuth(http.HandlerFunc(h.willHistory)))
	return withRequestID(mux)
}

type upsertWillRequest struct {
	State                 string   `json:"state"`
	DormancyPeriodDays    int      `json:"dormancyPeriodDays"`
	GracePeriodDays       int      `json:"gracePeriodDays"`
	PolicyVersionAccepted string   `json:"policyVersionAccepted"`
	ReleaseCategories     []string `json:"releaseCategories"`
}

func (h *Handler) createWill(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeWillRequest(w, r)
	if !ok {
		return
	}
	will, err := h.service.CreateWill(r.Context(), principalFromContext(r.Context()).UserID, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, willResponse(will))
}

func (h *Handler) getWill(w http.ResponseWriter, r *http.Request) {
	will, err := h.service.Will(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, willResponse(will))
}

func (h *Handler) updateWill(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeWillRequest(w, r)
	if !ok {
		return
	}
	will, err := h.service.UpdateWill(r.Context(), principalFromContext(r.Context()).UserID, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, willResponse(will))
}

func (h *Handler) deleteWill(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteWill(r.Context(), principalFromContext(r.Context()).UserID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) willHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.WillHistory(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(history))
	for _, item := range history {
		response = append(response, versionResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": response})
}

func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		principal, err := h.tokens.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(w, r, http.StatusConflict, "will_exists", "an active digital will already exists")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "resource not found")
	case isValidationError(err):
		writeError(w, r, http.StatusBadRequest, "validation_error", err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

func isValidationError(err error) bool {
	var validationError application.ValidationError
	return errors.As(err, &validationError)
}

func decodeWillRequest(w http.ResponseWriter, r *http.Request) (domain.UpsertWillInput, bool) {
	var request upsertWillRequest
	if !decode(w, r, &request) {
		return domain.UpsertWillInput{}, false
	}

	categories := make([]domain.ReleaseCategory, 0, len(request.ReleaseCategories))
	for _, category := range request.ReleaseCategories {
		categories = append(categories, domain.ReleaseCategory(strings.TrimSpace(category)))
	}

	return domain.UpsertWillInput{
		Status:                domain.WillStatus(strings.TrimSpace(request.State)),
		DormancyPeriodDays:    request.DormancyPeriodDays,
		GracePeriodDays:       request.GracePeriodDays,
		PolicyVersionAccepted: strings.TrimSpace(request.PolicyVersionAccepted),
		ReleaseCategories:     categories,
	}, true
}

func willResponse(will domain.DigitalWill) map[string]any {
	categories := make([]string, 0, len(will.ReleaseCategories))
	for _, category := range will.ReleaseCategories {
		categories = append(categories, string(category))
	}
	return map[string]any{
		"id":                    will.ID.String(),
		"userId":                will.UserID.String(),
		"state":                 string(will.Status),
		"version":               will.Version,
		"dormancyPeriodDays":    will.DormancyPeriodDays,
		"gracePeriodDays":       will.GracePeriodDays,
		"policyVersionAccepted": will.PolicyVersionAccepted,
		"consentAcceptedAt":     will.ConsentAcceptedAt.UTC().Format(time.RFC3339),
		"releaseCategories":     categories,
		"createdAt":             will.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":             will.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func versionResponse(version domain.WillVersion) map[string]any {
	categories := make([]string, 0, len(version.ReleaseCategories))
	for _, category := range version.ReleaseCategories {
		categories = append(categories, string(category))
	}
	return map[string]any{
		"id":                    version.ID.String(),
		"willId":                version.WillID.String(),
		"userId":                version.UserID.String(),
		"version":               version.Version,
		"state":                 string(version.Status),
		"dormancyPeriodDays":    version.DormancyPeriodDays,
		"gracePeriodDays":       version.GracePeriodDays,
		"policyVersionAccepted": version.PolicyVersionAccepted,
		"consentAcceptedAt":     version.ConsentAcceptedAt.UTC().Format(time.RFC3339),
		"releaseCategories":     categories,
		"createdAt":             version.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func decode(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return false
	}
	return true
}
