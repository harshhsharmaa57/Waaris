package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	service *application.Service
	tokens  *application.TokenVerifier
	ready   func(context.Context) error
}

func NewHandler(service *application.Service, tokens *application.TokenVerifier) *Handler {
	return NewHandlerWithReadiness(service, tokens, nil)
}

func NewHandlerWithReadiness(service *application.Service, tokens *application.TokenVerifier, ready func(context.Context) error) *Handler {
	return &Handler{service: service, tokens: tokens, ready: ready}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "enrollment"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if h.ready != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := h.ready(ctx); err != nil {
				writeError(w, r, http.StatusServiceUnavailable, "not_ready", "service dependencies are unavailable")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "enrollment"})
	})
	mux.Handle("POST /api/v1/will", h.requireAuth(http.HandlerFunc(h.createWill)))
	mux.Handle("GET /api/v1/will", h.requireAuth(http.HandlerFunc(h.getWill)))
	mux.Handle("PUT /api/v1/will", h.requireAuth(http.HandlerFunc(h.updateWill)))
	mux.Handle("DELETE /api/v1/will", h.requireAuth(http.HandlerFunc(h.deleteWill)))
	mux.Handle("GET /api/v1/will/history", h.requireAuth(http.HandlerFunc(h.willHistory)))
	mux.Handle("POST /api/v1/trustees", h.requireAuth(http.HandlerFunc(h.createTrustee)))
	mux.Handle("GET /api/v1/trustees", h.requireAuth(http.HandlerFunc(h.listTrustees)))
	mux.Handle("PUT /api/v1/trustees/{trusteeId}", h.requireAuth(http.HandlerFunc(h.updateTrustee)))
	mux.Handle("DELETE /api/v1/trustees/{trusteeId}", h.requireAuth(http.HandlerFunc(h.deleteTrustee)))
	mux.Handle("POST /api/v1/heartbeats", h.requireAuth(http.HandlerFunc(h.submitHeartbeat)))
	mux.Handle("GET /api/v1/heartbeats", h.requireAuth(http.HandlerFunc(h.getHeartbeatStatus)))
	mux.Handle("GET /api/v1/heartbeats/history", h.requireAuth(http.HandlerFunc(h.getHeartbeatHistory)))
	mux.Handle("GET /api/v1/verifications/pending", h.requireAuth(http.HandlerFunc(h.pendingVerifications)))
	mux.Handle("POST /api/v1/verifications/{requestId}/approve", h.requireAuth(http.HandlerFunc(h.approveVerification)))
	mux.Handle("POST /api/v1/verifications/{requestId}/reject", h.requireAuth(http.HandlerFunc(h.rejectVerification)))
	mux.Handle("POST /api/v1/verifications/{requestId}/abstain", h.requireAuth(http.HandlerFunc(h.abstainVerification)))
	mux.Handle("GET /api/v1/notifications/history", h.requireAuth(http.HandlerFunc(h.notificationHistory)))
	mux.Handle("GET /api/v1/audit/history", h.requireAuth(http.HandlerFunc(h.auditHistory)))
	return withRequestID(withHTTPProtection(mux))
}

type upsertWillRequest struct {
	State                 string   `json:"state"`
	DormancyPeriodDays    int      `json:"dormancyPeriodDays"`
	GracePeriodDays       int      `json:"gracePeriodDays"`
	PolicyVersionAccepted string   `json:"policyVersionAccepted"`
	ReleaseCategories     []string `json:"releaseCategories"`
}

type trusteeRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Relationship string `json:"relationship"`
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

func (h *Handler) createTrustee(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeTrusteeRequest(w, r)
	if !ok {
		return
	}
	trustee, err := h.service.CreateTrustee(r.Context(), principalFromContext(r.Context()).UserID, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, trusteeResponse(trustee))
}

func (h *Handler) listTrustees(w http.ResponseWriter, r *http.Request) {
	trustees, err := h.service.Trustees(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(trustees))
	for _, trustee := range trustees {
		response = append(response, trusteeResponse(trustee))
	}
	writeJSON(w, http.StatusOK, map[string]any{"trustees": response})
}

func (h *Handler) updateTrustee(w http.ResponseWriter, r *http.Request) {
	trusteeID, err := uuid.Parse(r.PathValue("trusteeId"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "trusteeId must be a valid UUID")
		return
	}
	input, ok := decodeTrusteeRequest(w, r)
	if !ok {
		return
	}
	trustee, err := h.service.UpdateTrustee(r.Context(), principalFromContext(r.Context()).UserID, trusteeID, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, trusteeResponse(trustee))
}

func (h *Handler) deleteTrustee(w http.ResponseWriter, r *http.Request) {
	trusteeID, err := uuid.Parse(r.PathValue("trusteeId"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "trusteeId must be a valid UUID")
		return
	}
	if err = h.service.DeleteTrustee(r.Context(), principalFromContext(r.Context()).UserID, trusteeID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) submitHeartbeat(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.SubmitHeartbeat(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, heartbeatStatusResponse(status))
}

func (h *Handler) getHeartbeatStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.HeartbeatStatus(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, heartbeatStatusResponse(status))
}

func (h *Handler) getHeartbeatHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.HeartbeatHistory(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(history))
	for _, item := range history {
		response = append(response, heartbeatResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": response})
}

func (h *Handler) pendingVerifications(w http.ResponseWriter, r *http.Request) {
	requests, err := h.service.PendingVerifications(r.Context(), principalFromContext(r.Context()).Email)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(requests))
	for _, item := range requests {
		response = append(response, verificationRequestResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"pending": response})
}

func (h *Handler) approveVerification(w http.ResponseWriter, r *http.Request) {
	h.respondVerification(w, r, h.service.ApproveVerification)
}

func (h *Handler) rejectVerification(w http.ResponseWriter, r *http.Request) {
	h.respondVerification(w, r, h.service.RejectVerification)
}

func (h *Handler) abstainVerification(w http.ResponseWriter, r *http.Request) {
	h.respondVerification(w, r, h.service.AbstainVerification)
}

func (h *Handler) respondVerification(w http.ResponseWriter, r *http.Request, action func(context.Context, string, uuid.UUID, uuid.UUID) error) {
	requestID, err := uuid.Parse(r.PathValue("requestId"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "requestId must be a valid UUID")
		return
	}
	principal := principalFromContext(r.Context())
	if err = action(r.Context(), principal.Email, principal.UserID, requestID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) notificationHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.NotificationHistory(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(history))
	for _, item := range history {
		response = append(response, notificationResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": response})
}

func (h *Handler) auditHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.AuditHistory(r.Context(), principalFromContext(r.Context()).UserID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	response := make([]map[string]any, 0, len(history))
	for _, item := range history {
		response = append(response, auditResponse(item))
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
	case errors.Is(err, domain.ErrDuplicateTrustee):
		writeError(w, r, http.StatusConflict, "duplicate_trustee", "trustee is already configured")
	case errors.Is(err, domain.ErrSelfTrustee):
		writeError(w, r, http.StatusBadRequest, "self_trustee", "will owner cannot be a trustee")
	case errors.Is(err, domain.ErrPublishedRequiresTrustee):
		writeError(w, r, http.StatusConflict, "published_requires_trustee", "published wills require at least one trustee")
	case errors.Is(err, domain.ErrWillNotEditable):
		writeError(w, r, http.StatusConflict, "will_not_editable", "will configuration is locked while verification is in progress")
	case errors.Is(err, domain.ErrVerificationNotPending):
		writeError(w, r, http.StatusConflict, "verification_not_pending", "verification request is not pending")
	case errors.Is(err, domain.ErrTrusteeNotAssigned):
		writeError(w, r, http.StatusForbidden, "trustee_not_assigned", "current user is not assigned as a trustee for this request")
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

func decodeTrusteeRequest(w http.ResponseWriter, r *http.Request) (domain.TrusteeInput, bool) {
	var request trusteeRequest
	if !decode(w, r, &request) {
		return domain.TrusteeInput{}, false
	}
	return domain.TrusteeInput{Name: request.Name, Email: request.Email, Relationship: request.Relationship}, true
}

func willResponse(will domain.DigitalWill) map[string]any {
	categories := make([]string, 0, len(will.ReleaseCategories))
	for _, category := range will.ReleaseCategories {
		categories = append(categories, string(category))
	}
	return map[string]any{
		"id":                           will.ID.String(),
		"userId":                       will.UserID.String(),
		"state":                        string(will.Status),
		"lifecycleState":               string(will.LifecycleState),
		"version":                      will.Version,
		"dormancyPeriodDays":           will.DormancyPeriodDays,
		"gracePeriodDays":              will.GracePeriodDays,
		"policyVersionAccepted":        will.PolicyVersionAccepted,
		"consentAcceptedAt":            will.ConsentAcceptedAt.UTC().Format(time.RFC3339),
		"lastHeartbeatAt":              formatTimePtr(will.LastHeartbeatAt),
		"pendingVerificationStartedAt": formatTimePtr(will.PendingVerificationStartedAt),
		"gracePeriodStartedAt":         formatTimePtr(will.GracePeriodStartedAt),
		"readyForExecutionAt":          formatTimePtr(will.ReadyForExecutionAt),
		"releaseCategories":            categories,
		"createdAt":                    will.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":                    will.UpdatedAt.UTC().Format(time.RFC3339),
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

func trusteeResponse(trustee domain.Trustee) map[string]any {
	return map[string]any{
		"id":           trustee.ID.String(),
		"willId":       trustee.WillID.String(),
		"name":         trustee.Name,
		"email":        trustee.Email,
		"relationship": trustee.Relationship,
		"createdAt":    trustee.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":    trustee.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func heartbeatStatusResponse(status domain.HeartbeatStatus) map[string]any {
	return map[string]any{
		"willId":                       status.WillID.String(),
		"lifecycleState":               string(status.LifecycleState),
		"lastHeartbeatAt":              formatTimePtr(status.LastHeartbeatAt),
		"pendingVerificationStartedAt": formatTimePtr(status.PendingVerificationStartedAt),
		"gracePeriodStartedAt":         formatTimePtr(status.GracePeriodStartedAt),
		"readyForExecutionAt":          formatTimePtr(status.ReadyForExecutionAt),
	}
}

func heartbeatResponse(item domain.Heartbeat) map[string]any {
	return map[string]any{
		"id":         item.ID.String(),
		"willId":     item.WillID.String(),
		"userId":     item.UserID.String(),
		"source":     item.Source,
		"occurredAt": item.OccurredAt.UTC().Format(time.RFC3339),
		"createdAt":  item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func verificationRequestResponse(item domain.VerificationRequest) map[string]any {
	response := map[string]any{
		"id":                item.ID.String(),
		"willId":            item.WillID.String(),
		"userId":            item.UserID.String(),
		"thresholdRequired": item.ThresholdRequired,
		"status":            item.Status,
		"createdAt":         item.CreatedAt.UTC().Format(time.RFC3339),
		"ownerEmail":        item.OwnerEmail,
		"ownerDisplayName":  item.OwnerDisplayName,
		"trustee":           trusteeResponse(item.Trustee),
	}
	if item.ResolvedAt != nil {
		response["resolvedAt"] = item.ResolvedAt.UTC().Format(time.RFC3339)
	}
	if item.LatestDecision != nil {
		response["latestDecision"] = string(*item.LatestDecision)
	}
	return response
}

func notificationResponse(item domain.Notification) map[string]any {
	response := map[string]any{
		"id":             item.ID.String(),
		"willId":         item.WillID.String(),
		"userId":         item.UserID.String(),
		"eventType":      item.EventType,
		"channel":        item.Channel,
		"recipientName":  item.RecipientName,
		"recipientEmail": item.RecipientEmail,
		"subject":        item.Subject,
		"body":           item.Body,
		"status":         item.Status,
		"queuedAt":       item.QueuedAt.UTC().Format(time.RFC3339),
	}
	if item.TrusteeID != nil {
		response["trusteeId"] = item.TrusteeID.String()
	}
	if item.SentAt != nil {
		response["sentAt"] = item.SentAt.UTC().Format(time.RFC3339)
	}
	if item.FailureMessage != "" {
		response["failureMessage"] = item.FailureMessage
	}
	return response
}

func auditResponse(item domain.AuditEvent) map[string]any {
	response := map[string]any{
		"id":            item.ID.String(),
		"actorType":     item.ActorType,
		"actorId":       item.ActorID,
		"eventType":     item.EventType,
		"correlationId": item.CorrelationID,
		"details":       item.Details,
		"occurredAt":    item.OccurredAt.UTC().Format(time.RFC3339),
	}
	if item.UserID != nil {
		response["userId"] = item.UserID.String()
	}
	if item.WillID != nil {
		response["willId"] = item.WillID.String()
	}
	return response
}

func decode(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return false
	}
	return true
}

func formatTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
