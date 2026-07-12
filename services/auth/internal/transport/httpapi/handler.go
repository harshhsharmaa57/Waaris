package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/waaris/waaris/services/auth/internal/application"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

const maxBodyBytes = 1 << 20

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth"})
	})
	mux.HandleFunc("POST /v1/auth/register", h.register)
	mux.HandleFunc("POST /v1/auth/login", h.login)
	mux.HandleFunc("POST /v1/auth/refresh", h.refresh)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
	mux.Handle("GET /v1/users/me", h.requireAuth(http.HandlerFunc(h.profile)))
	mux.Handle("PATCH /v1/users/me", h.requireAuth(http.HandlerFunc(h.updateProfile)))
	mux.Handle("DELETE /v1/users/me", h.requireAuth(http.HandlerFunc(h.deleteProfile)))
	return withRequestID(mux)
}

type credentialsRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}
type profileRequest struct {
	DisplayName string `json:"displayName"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decode(w, r, &request) {
		return
	}
	if err := validateCredentials(request.Email, request.Password, request.DisplayName); err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	session, err := h.service.Register(r.Context(), request.Email, request.Password, request.DisplayName)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, sessionResponse(session))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decode(w, r, &request) {
		return
	}
	if err := validateEmailAndPassword(request.Email, request.Password); err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	session, err := h.service.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse(session))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if !decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.RefreshToken) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_error", "refreshToken is required")
		return
	}
	session, err := h.service.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse(session))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if !decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.RefreshToken) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_error", "refreshToken is required")
		return
	}
	if err := h.service.Logout(r.Context(), request.RefreshToken); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Profile(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var request profileRequest
	if !decode(w, r, &request) {
		return
	}
	if len(strings.TrimSpace(request.DisplayName)) > 100 {
		writeError(w, r, http.StatusBadRequest, "validation_error", "displayName must not exceed 100 characters")
		return
	}
	user, err := h.service.UpdateProfile(r.Context(), principalFromContext(r.Context()).UserID, request.DisplayName)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (h *Handler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteProfile(r.Context(), principalFromContext(r.Context()).UserID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		principal, err := h.service.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		writeError(w, r, http.StatusConflict, "email_taken", "email is already registered")
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidRefreshToken):
		writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "resource not found")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

func validateCredentials(email, password, displayName string) error {
	if err := validateEmailAndPassword(email, password); err != nil {
		return err
	}
	if len(strings.TrimSpace(displayName)) > 100 {
		return errors.New("displayName must not exceed 100 characters")
	}
	return nil
}
func validateEmailAndPassword(email, password string) error {
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return errors.New("email must be valid")
	}
	if len(password) < 12 || len(password) > 128 {
		return errors.New("password must be between 12 and 128 characters")
	}
	return nil
}

func sessionResponse(session application.Session) map[string]any {
	return map[string]any{"user": userResponse(session.User), "accessToken": session.AccessToken, "refreshToken": session.RefreshToken, "accessTokenExpiresAt": session.ExpiresAt.UTC().Format(time.RFC3339)}
}
func userResponse(user domain.User) map[string]any {
	return map[string]any{"id": user.ID.String(), "email": user.Email, "displayName": user.DisplayName, "createdAt": user.CreatedAt.UTC().Format(time.RFC3339), "updatedAt": user.UpdatedAt.UTC().Format(time.RFC3339)}
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
