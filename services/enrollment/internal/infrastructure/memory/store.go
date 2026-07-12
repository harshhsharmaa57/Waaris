package memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type storedWill struct {
	current domain.DigitalWill
	history []domain.WillVersion
	deleted bool
}

type verificationRound struct {
	request   domain.VerificationRequest
	responses []domain.VerificationResponse
}

type Store struct {
	mu            sync.Mutex
	wills         map[uuid.UUID]storedWill
	trustees      map[uuid.UUID][]domain.Trustee
	heartbeats    map[uuid.UUID][]domain.Heartbeat
	verifications map[uuid.UUID]verificationRound
	notifications []domain.Notification
	audit         []domain.AuditEvent
}

func New() *Store {
	return &Store{
		wills:         map[uuid.UUID]storedWill{},
		trustees:      map[uuid.UUID][]domain.Trustee{},
		heartbeats:    map[uuid.UUID][]domain.Heartbeat{},
		verifications: map[uuid.UUID]verificationRound{},
	}
}

func (s *Store) CreateWill(_ context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.wills[userID]; ok && !existing.deleted {
		return domain.DigitalWill{}, domain.ErrAlreadyExists
	}

	willID := uuid.New()
	versionID := uuid.New()
	current := domain.DigitalWill{
		ID:                    willID,
		UserID:                userID,
		OwnerEmail:            userID.String() + "@example.com",
		Status:                input.Status,
		LifecycleState:        domain.LifecycleActive,
		Version:               1,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	history := []domain.WillVersion{{
		ID:                    versionID,
		WillID:                willID,
		UserID:                userID,
		Version:               1,
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
	}}
	s.wills[userID] = storedWill{current: current, history: history}
	s.appendAuditLocked(current, "user", userID.String(), "will_created", "{}", now)
	return cloneWill(current), nil
}

func (s *Store) WillByUser(_ context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getWillLocked(userID)
}

func (s *Store) UpdateWill(_ context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.DigitalWill{}, domain.ErrNotFound
	}
	value.current.Version++
	value.current.Status = input.Status
	value.current.DormancyPeriodDays = input.DormancyPeriodDays
	value.current.GracePeriodDays = input.GracePeriodDays
	value.current.PolicyVersionAccepted = input.PolicyVersionAccepted
	value.current.ConsentAcceptedAt = now
	value.current.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	value.current.UpdatedAt = now
	value.history = append(value.history, domain.WillVersion{
		ID:                    uuid.New(),
		WillID:                value.current.ID,
		UserID:                userID,
		Version:               value.current.Version,
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: input.PolicyVersionAccepted,
		ConsentAcceptedAt:     now,
		ReleaseCategories:     domain.CloneCategories(input.ReleaseCategories),
		CreatedAt:             now,
	})
	s.wills[userID] = value
	s.appendAuditLocked(value.current, "user", userID.String(), "will_updated", "{}", now)
	return cloneWill(value.current), nil
}

func (s *Store) DeleteWill(_ context.Context, userID uuid.UUID, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.ErrNotFound
	}
	value.deleted = true
	value.current.UpdatedAt = now
	s.wills[userID] = value
	s.appendAuditLocked(value.current, "user", userID.String(), "will_deleted", "{}", now)
	return nil
}

func (s *Store) WillHistory(_ context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return nil, domain.ErrNotFound
	}
	result := make([]domain.WillVersion, 0, len(value.history))
	for index := len(value.history) - 1; index >= 0; index-- {
		item := value.history[index]
		item.ReleaseCategories = domain.CloneCategories(item.ReleaseCategories)
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) TrusteeCount(_ context.Context, userID uuid.UUID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return 0, err
	}
	return len(s.trustees[will.ID]), nil
}

func (s *Store) CreateTrustee(_ context.Context, userID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return domain.Trustee{}, err
	}
	for _, trustee := range s.trustees[will.ID] {
		if strings.EqualFold(trustee.Email, input.Email) {
			return domain.Trustee{}, domain.ErrDuplicateTrustee
		}
	}
	trustee := domain.Trustee{ID: uuid.New(), WillID: will.ID, UserID: userID, Name: input.Name, Email: input.Email, Relationship: input.Relationship, CreatedAt: now, UpdatedAt: now}
	s.trustees[will.ID] = append(s.trustees[will.ID], trustee)
	s.appendAuditLocked(will, "user", userID.String(), "trustee_created", fmt.Sprintf(`{"trusteeId":"%s"}`, trustee.ID), now)
	return trustee, nil
}

func (s *Store) TrusteesByUser(_ context.Context, userID uuid.UUID) ([]domain.Trustee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return nil, err
	}
	result := slices.Clone(s.trustees[will.ID])
	slices.SortFunc(result, func(a, b domain.Trustee) int { return strings.Compare(a.Email, b.Email) })
	return result, nil
}

func (s *Store) UpdateTrustee(_ context.Context, userID uuid.UUID, trusteeID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return domain.Trustee{}, err
	}
	trustees := s.trustees[will.ID]
	for index, trustee := range trustees {
		if trustee.ID != trusteeID {
			if strings.EqualFold(trustee.Email, input.Email) {
				return domain.Trustee{}, domain.ErrDuplicateTrustee
			}
			continue
		}
		trustee.Name = input.Name
		trustee.Email = input.Email
		trustee.Relationship = input.Relationship
		trustee.UpdatedAt = now
		trustees[index] = trustee
		s.trustees[will.ID] = trustees
		s.appendAuditLocked(will, "user", userID.String(), "trustee_updated", fmt.Sprintf(`{"trusteeId":"%s"}`, trustee.ID), now)
		return trustee, nil
	}
	return domain.Trustee{}, domain.ErrNotFound
}

func (s *Store) DeleteTrustee(_ context.Context, userID uuid.UUID, trusteeID uuid.UUID, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return err
	}
	trustees := s.trustees[will.ID]
	for index, trustee := range trustees {
		if trustee.ID != trusteeID {
			continue
		}
		s.trustees[will.ID] = append(trustees[:index], trustees[index+1:]...)
		s.appendAuditLocked(will, "user", userID.String(), "trustee_deleted", fmt.Sprintf(`{"trusteeId":"%s"}`, trustee.ID), now)
		return nil
	}
	return domain.ErrNotFound
}

func (s *Store) SubmitHeartbeat(_ context.Context, userID uuid.UUID, source string, now time.Time) (domain.HeartbeatStatus, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.HeartbeatStatus{}, false, domain.ErrNotFound
	}
	heartbeat := domain.Heartbeat{ID: uuid.New(), WillID: value.current.ID, UserID: userID, Source: source, OccurredAt: now, CreatedAt: now}
	s.heartbeats[value.current.ID] = append([]domain.Heartbeat{heartbeat}, s.heartbeats[value.current.ID]...)
	value.current.LastHeartbeatAt = ptrTime(now)
	restored := value.current.LifecycleState != domain.LifecycleActive
	if restored {
		value.current.LifecycleState = domain.LifecycleActive
		value.current.PendingVerificationStartedAt = nil
		value.current.GracePeriodStartedAt = nil
		value.current.ReadyForExecutionAt = nil
		s.cancelPendingRoundsLocked(value.current.ID, now)
		s.appendAuditLocked(value.current, "user", userID.String(), "liveness_restored", "{}", now)
	}
	value.current.UpdatedAt = now
	s.wills[userID] = value
	s.appendAuditLocked(value.current, "user", userID.String(), "heartbeat_submitted", "{}", now)
	return heartbeatStatusFromWill(value.current), restored, nil
}

func (s *Store) HeartbeatStatus(_ context.Context, userID uuid.UUID) (domain.HeartbeatStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return domain.HeartbeatStatus{}, err
	}
	return heartbeatStatusFromWill(will), nil
}

func (s *Store) HeartbeatHistory(_ context.Context, userID uuid.UUID) ([]domain.Heartbeat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	will, err := s.getWillLocked(userID)
	if err != nil {
		return nil, err
	}
	return slices.Clone(s.heartbeats[will.ID]), nil
}

func (s *Store) TransitionOverdueWills(_ context.Context, now time.Time) ([]application.LifecycleEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var events []application.LifecycleEvent
	for userID, value := range s.wills {
		if value.deleted || value.current.Status != domain.StatusPublished {
			continue
		}
		switch value.current.LifecycleState {
		case domain.LifecycleActive:
			reference := value.current.UpdatedAt
			if value.current.LastHeartbeatAt != nil {
				reference = *value.current.LastHeartbeatAt
			}
			if now.Before(reference.Add(time.Duration(value.current.DormancyPeriodDays) * 24 * time.Hour)) {
				continue
			}
			trustees := slices.Clone(s.trustees[value.current.ID])
			if len(trustees) == 0 {
				continue
			}
			threshold := (len(trustees) / 2) + 1
			requestID := uuid.New()
			value.current.LifecycleState = domain.LifecyclePendingVerification
			value.current.PendingVerificationStartedAt = ptrTime(now)
			value.current.UpdatedAt = now
			s.wills[userID] = value
			s.verifications[requestID] = verificationRound{request: domain.VerificationRequest{ID: requestID, WillID: value.current.ID, UserID: userID, ThresholdRequired: threshold, Status: "pending", CreatedAt: now}}
			s.appendAuditLocked(value.current, "system", "lifecycle-tick", "dormancy_detected", "{}", now)
			s.appendAuditLocked(value.current, "system", "lifecycle-tick", "verification_started", fmt.Sprintf(`{"requestId":"%s"}`, requestID), now)
			events = append(events, application.LifecycleEvent{
				UserID:              userID,
				WillID:              value.current.ID,
				OwnerEmail:          value.current.OwnerEmail,
				OwnerDisplayName:    value.current.OwnerDisplayName,
				ThresholdRequired:   threshold,
				Trustees:            trustees,
				TransitionedAt:      now,
				PreviousState:       domain.LifecycleActive,
				CurrentState:        domain.LifecyclePendingVerification,
				GracePeriodDays:     value.current.GracePeriodDays,
				VerificationRequest: &requestID,
			})
		case domain.LifecycleGracePeriod:
			if value.current.GracePeriodStartedAt == nil {
				continue
			}
			if now.Before(value.current.GracePeriodStartedAt.Add(time.Duration(value.current.GracePeriodDays) * 24 * time.Hour)) {
				continue
			}
			value.current.LifecycleState = domain.LifecycleReadyForExecution
			value.current.ReadyForExecutionAt = ptrTime(now)
			value.current.UpdatedAt = now
			s.wills[userID] = value
			s.appendAuditLocked(value.current, "system", "lifecycle-tick", "ready_for_execution", "{}", now)
			events = append(events, application.LifecycleEvent{
				UserID:           userID,
				WillID:           value.current.ID,
				OwnerEmail:       value.current.OwnerEmail,
				OwnerDisplayName: value.current.OwnerDisplayName,
				TransitionedAt:   now,
				PreviousState:    domain.LifecycleGracePeriod,
				CurrentState:     domain.LifecycleReadyForExecution,
			})
		}
	}
	return events, nil
}

func (s *Store) PendingVerifications(_ context.Context, trusteeEmail string) ([]domain.VerificationRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.VerificationRequest
	for _, round := range s.verifications {
		if round.request.Status != "pending" {
			continue
		}
		owner := s.wills[round.request.UserID].current
		for _, trustee := range s.trustees[round.request.WillID] {
			if !strings.EqualFold(trustee.Email, trusteeEmail) {
				continue
			}
			request := round.request
			request.Trustee = trustee
			request.OwnerEmail = owner.OwnerEmail
			request.OwnerDisplayName = owner.OwnerDisplayName
			if decision, ok := latestDecision(round.responses, trustee.ID); ok {
				request.LatestDecision = &decision
			}
			result = append(result, request)
		}
	}
	slices.SortFunc(result, func(a, b domain.VerificationRequest) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return result, nil
}

func (s *Store) RecordVerificationDecision(_ context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID, decision domain.VerificationDecision, now time.Time) (application.VerificationDecisionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	round, ok := s.verifications[requestID]
	if !ok {
		return application.VerificationDecisionResult{}, domain.ErrNotFound
	}
	if round.request.Status != "pending" {
		return application.VerificationDecisionResult{}, domain.ErrVerificationNotPending
	}
	owner, ok := s.wills[round.request.UserID]
	if !ok || owner.deleted {
		return application.VerificationDecisionResult{}, domain.ErrNotFound
	}
	var actorTrustee domain.Trustee
	found := false
	trustees := s.trustees[round.request.WillID]
	for _, trustee := range trustees {
		if strings.EqualFold(trustee.Email, trusteeEmail) {
			actorTrustee = trustee
			found = true
			break
		}
	}
	if !found {
		return application.VerificationDecisionResult{}, domain.ErrTrusteeNotAssigned
	}
	response := domain.VerificationResponse{ID: uuid.New(), RequestID: requestID, WillID: round.request.WillID, TrusteeID: actorTrustee.ID, ActorUserID: actorUserID, Decision: decision, RespondedAt: now}
	round.responses = append(round.responses, response)
	result := application.VerificationDecisionResult{
		UserID:            round.request.UserID,
		WillID:            round.request.WillID,
		OwnerEmail:        owner.current.OwnerEmail,
		OwnerDisplayName:  owner.current.OwnerDisplayName,
		Trustees:          slices.Clone(trustees),
		RequestID:         requestID,
		Decision:          decision,
		ActorTrustee:      actorTrustee,
		ThresholdRequired: round.request.ThresholdRequired,
	}
	approvals := latestApprovalCount(round.responses)
	result.Approvals = approvals
	if approvals >= round.request.ThresholdRequired {
		owner.current.LifecycleState = domain.LifecycleGracePeriod
		owner.current.GracePeriodStartedAt = ptrTime(now)
		owner.current.UpdatedAt = now
		s.wills[round.request.UserID] = owner
		round.request.Status = "resolved"
		round.request.ResolvedAt = ptrTime(now)
		state := domain.LifecycleGracePeriod
		result.TransitionedTo = &state
		s.appendAuditLocked(owner.current, "trustee", actorTrustee.Email, "trustee_response_recorded", fmt.Sprintf(`{"requestId":"%s","decision":"%s"}`, requestID, decision), now)
		s.appendAuditLocked(owner.current, "system", "verification-threshold", "grace_period_started", fmt.Sprintf(`{"requestId":"%s"}`, requestID), now)
	} else {
		s.appendAuditLocked(owner.current, "trustee", actorTrustee.Email, "trustee_response_recorded", fmt.Sprintf(`{"requestId":"%s","decision":"%s"}`, requestID, decision), now)
	}
	s.verifications[requestID] = round
	return result, nil
}

func (s *Store) NotificationHistory(_ context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.Notification
	for _, notification := range s.notifications {
		if notification.UserID == userID {
			result = append(result, notification)
		}
	}
	slices.SortFunc(result, func(a, b domain.Notification) int { return b.QueuedAt.Compare(a.QueuedAt) })
	return result, nil
}

func (s *Store) AuditHistory(_ context.Context, userID uuid.UUID) ([]domain.AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.AuditEvent
	for _, event := range s.audit {
		if event.UserID != nil && *event.UserID == userID {
			result = append(result, event)
		}
	}
	slices.SortFunc(result, func(a, b domain.AuditEvent) int { return b.OccurredAt.Compare(a.OccurredAt) })
	return result, nil
}

func (s *Store) Enqueue(_ context.Context, notifications []domain.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, notifications...)
	for _, notification := range notifications {
		will := s.findWillByIDLocked(notification.WillID)
		if will != nil {
			s.appendAuditLocked(*will, "system", "notification-queue", "notification_queued", fmt.Sprintf(`{"notificationId":"%s","eventType":"%s"}`, notification.ID, notification.EventType), notification.QueuedAt)
		}
	}
	return nil
}

func (s *Store) Pending(_ context.Context, limit int) ([]domain.Notification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.Notification
	for _, notification := range s.notifications {
		if notification.Status == "queued" {
			result = append(result, notification)
		}
	}
	slices.SortFunc(result, func(a, b domain.Notification) int { return a.QueuedAt.Compare(b.QueuedAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *Store) MarkSent(_ context.Context, notificationID uuid.UUID, sentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, notification := range s.notifications {
		if notification.ID != notificationID {
			continue
		}
		notification.Status = "sent"
		notification.SentAt = ptrTime(sentAt)
		s.notifications[index] = notification
		if will := s.findWillByIDLocked(notification.WillID); will != nil {
			s.appendAuditLocked(*will, "system", "notification-dispatch", "notification_sent", fmt.Sprintf(`{"notificationId":"%s"}`, notification.ID), sentAt)
		}
		return nil
	}
	return domain.ErrNotFound
}

func (s *Store) MarkFailed(_ context.Context, notificationID uuid.UUID, message string, failedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, notification := range s.notifications {
		if notification.ID != notificationID {
			continue
		}
		notification.Status = "failed"
		notification.FailureMessage = message
		s.notifications[index] = notification
		if will := s.findWillByIDLocked(notification.WillID); will != nil {
			s.appendAuditLocked(*will, "system", "notification-dispatch", "notification_failed", fmt.Sprintf(`{"notificationId":"%s"}`, notification.ID), failedAt)
		}
		return nil
	}
	return domain.ErrNotFound
}

func (s *Store) getWillLocked(userID uuid.UUID) (domain.DigitalWill, error) {
	value, ok := s.wills[userID]
	if !ok || value.deleted {
		return domain.DigitalWill{}, domain.ErrNotFound
	}
	return cloneWill(value.current), nil
}

func (s *Store) cancelPendingRoundsLocked(willID uuid.UUID, now time.Time) {
	for requestID, round := range s.verifications {
		if round.request.WillID == willID && round.request.Status == "pending" {
			round.request.Status = "cancelled"
			round.request.ResolvedAt = ptrTime(now)
			s.verifications[requestID] = round
		}
	}
}

func (s *Store) appendAuditLocked(will domain.DigitalWill, actorType, actorID, eventType, details string, now time.Time) {
	userID := will.UserID
	willID := will.ID
	s.audit = append(s.audit, domain.AuditEvent{
		ID:            uuid.New(),
		UserID:        &userID,
		WillID:        &willID,
		ActorType:     actorType,
		ActorID:       actorID,
		EventType:     eventType,
		CorrelationID: "",
		Details:       details,
		OccurredAt:    now,
	})
}

func latestApprovalCount(responses []domain.VerificationResponse) int {
	latest := map[uuid.UUID]domain.VerificationDecision{}
	for _, response := range responses {
		latest[response.TrusteeID] = response.Decision
	}
	count := 0
	for _, decision := range latest {
		if decision == domain.DecisionApprove {
			count++
		}
	}
	return count
}

func latestDecision(responses []domain.VerificationResponse, trusteeID uuid.UUID) (domain.VerificationDecision, bool) {
	for index := len(responses) - 1; index >= 0; index-- {
		if responses[index].TrusteeID == trusteeID {
			return responses[index].Decision, true
		}
	}
	return "", false
}

func cloneWill(value domain.DigitalWill) domain.DigitalWill {
	value.ReleaseCategories = domain.CloneCategories(value.ReleaseCategories)
	return value
}

func heartbeatStatusFromWill(value domain.DigitalWill) domain.HeartbeatStatus {
	return domain.HeartbeatStatus{
		WillID:                       value.ID,
		LifecycleState:               value.LifecycleState,
		LastHeartbeatAt:              value.LastHeartbeatAt,
		PendingVerificationStartedAt: value.PendingVerificationStartedAt,
		GracePeriodStartedAt:         value.GracePeriodStartedAt,
		ReadyForExecutionAt:          value.ReadyForExecutionAt,
	}
}

func ptrTime(value time.Time) *time.Time {
	result := value.UTC()
	return &result
}

func (s *Store) findWillByIDLocked(willID uuid.UUID) *domain.DigitalWill {
	for _, stored := range s.wills {
		if stored.current.ID == willID && !stored.deleted {
			copy := cloneWill(stored.current)
			return &copy
		}
	}
	return nil
}
