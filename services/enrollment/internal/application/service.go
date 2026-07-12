package application

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

var (
	allowedCategories = map[domain.ReleaseCategory]struct{}{
		domain.CategoryFinancial:          {},
		domain.CategoryPrivate:            {},
		domain.CategoryCommunityShareable: {},
	}
	allowedLifecycleStates = map[domain.LifecycleState]struct{}{
		domain.LifecycleActive:              {},
		domain.LifecyclePendingVerification: {},
		domain.LifecycleGracePeriod:         {},
		domain.LifecycleReadyForExecution:   {},
	}
)

type ValidationError struct{ message string }

func (e ValidationError) Error() string { return e.message }

type Service struct {
	store    Store
	queue    NotificationQueue
	notifier Notifier
	now      func() time.Time
}

func NewService(store Store, queue NotificationQueue, notifier Notifier) *Service {
	return NewServiceWithClock(store, queue, notifier, time.Now)
}

func NewServiceWithClock(store Store, queue NotificationQueue, notifier Notifier, now func() time.Time) *Service {
	return &Service{store: store, queue: queue, notifier: notifier, now: now}
}

func (s *Service) CreateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput) (domain.DigitalWill, error) {
	normalized, err := normalizeWillInput(input)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	if normalized.Status == domain.StatusPublished {
		trusteeCount, err := s.store.TrusteeCount(ctx, userID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.DigitalWill{}, domain.ErrPublishedRequiresTrustee
		}
		if err != nil {
			return domain.DigitalWill{}, err
		}
		if trusteeCount == 0 {
			return domain.DigitalWill{}, domain.ErrPublishedRequiresTrustee
		}
	}
	return s.store.CreateWill(ctx, userID, normalized, s.now().UTC())
}

func (s *Service) Will(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	return s.store.WillByUser(ctx, userID)
}

func (s *Service) UpdateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput) (domain.DigitalWill, error) {
	if err := s.requireEditableWill(ctx, userID); err != nil {
		return domain.DigitalWill{}, err
	}
	normalized, err := normalizeWillInput(input)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	if normalized.Status == domain.StatusPublished {
		trusteeCount, err := s.store.TrusteeCount(ctx, userID)
		if err != nil {
			return domain.DigitalWill{}, err
		}
		if trusteeCount == 0 {
			return domain.DigitalWill{}, domain.ErrPublishedRequiresTrustee
		}
	}
	return s.store.UpdateWill(ctx, userID, normalized, s.now().UTC())
}

func (s *Service) DeleteWill(ctx context.Context, userID uuid.UUID) error {
	return s.store.DeleteWill(ctx, userID, s.now().UTC())
}

func (s *Service) WillHistory(ctx context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	return s.store.WillHistory(ctx, userID)
}

func (s *Service) CreateTrustee(ctx context.Context, userID uuid.UUID, input domain.TrusteeInput) (domain.Trustee, error) {
	normalized, err := normalizeTrusteeInput(input)
	if err != nil {
		return domain.Trustee{}, err
	}
	will, err := s.editableWill(ctx, userID)
	if err != nil {
		return domain.Trustee{}, err
	}
	if strings.EqualFold(normalized.Email, will.OwnerEmail) {
		return domain.Trustee{}, domain.ErrSelfTrustee
	}
	return s.store.CreateTrustee(ctx, userID, normalized, s.now().UTC())
}

func (s *Service) Trustees(ctx context.Context, userID uuid.UUID) ([]domain.Trustee, error) {
	return s.store.TrusteesByUser(ctx, userID)
}

func (s *Service) UpdateTrustee(ctx context.Context, userID, trusteeID uuid.UUID, input domain.TrusteeInput) (domain.Trustee, error) {
	normalized, err := normalizeTrusteeInput(input)
	if err != nil {
		return domain.Trustee{}, err
	}
	will, err := s.editableWill(ctx, userID)
	if err != nil {
		return domain.Trustee{}, err
	}
	if strings.EqualFold(normalized.Email, will.OwnerEmail) {
		return domain.Trustee{}, domain.ErrSelfTrustee
	}
	return s.store.UpdateTrustee(ctx, userID, trusteeID, normalized, s.now().UTC())
}

func (s *Service) DeleteTrustee(ctx context.Context, userID, trusteeID uuid.UUID) error {
	will, err := s.editableWill(ctx, userID)
	if err != nil {
		return err
	}
	if will.Status == domain.StatusPublished {
		trusteeCount, err := s.store.TrusteeCount(ctx, userID)
		if err != nil {
			return err
		}
		if trusteeCount <= 1 {
			return domain.ErrPublishedRequiresTrustee
		}
	}
	return s.store.DeleteTrustee(ctx, userID, trusteeID, s.now().UTC())
}

func (s *Service) requireEditableWill(ctx context.Context, userID uuid.UUID) error {
	_, err := s.editableWill(ctx, userID)
	return err
}

func (s *Service) editableWill(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	will, err := s.store.WillByUser(ctx, userID)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	if will.LifecycleState != domain.LifecycleActive {
		return domain.DigitalWill{}, domain.ErrWillNotEditable
	}
	return will, nil
}

func (s *Service) SubmitHeartbeat(ctx context.Context, userID uuid.UUID) (domain.HeartbeatStatus, error) {
	status, restored, err := s.store.SubmitHeartbeat(ctx, userID, "api", s.now().UTC())
	if err != nil {
		return domain.HeartbeatStatus{}, err
	}
	if restored {
		will, err := s.store.WillByUser(ctx, userID)
		if err != nil {
			return domain.HeartbeatStatus{}, err
		}
		trustees, err := s.store.TrusteesByUser(ctx, userID)
		if err != nil {
			return domain.HeartbeatStatus{}, err
		}
		notifications := recoveryNotifications(will, trustees, s.now().UTC())
		if err = s.queue.Enqueue(ctx, notifications); err != nil {
			return domain.HeartbeatStatus{}, err
		}
		if err = s.deliverPending(ctx, 20); err != nil {
			return domain.HeartbeatStatus{}, err
		}
	}
	return status, nil
}

func (s *Service) HeartbeatStatus(ctx context.Context, userID uuid.UUID) (domain.HeartbeatStatus, error) {
	return s.store.HeartbeatStatus(ctx, userID)
}

func (s *Service) HeartbeatHistory(ctx context.Context, userID uuid.UUID) ([]domain.Heartbeat, error) {
	return s.store.HeartbeatHistory(ctx, userID)
}

func (s *Service) PendingVerifications(ctx context.Context, trusteeEmail string) ([]domain.VerificationRequest, error) {
	return s.store.PendingVerifications(ctx, strings.ToLower(strings.TrimSpace(trusteeEmail)))
}

func (s *Service) ApproveVerification(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID) error {
	return s.respondVerification(ctx, trusteeEmail, actorUserID, requestID, domain.DecisionApprove)
}

func (s *Service) RejectVerification(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID) error {
	return s.respondVerification(ctx, trusteeEmail, actorUserID, requestID, domain.DecisionReject)
}

func (s *Service) AbstainVerification(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID) error {
	return s.respondVerification(ctx, trusteeEmail, actorUserID, requestID, domain.DecisionAbstain)
}

func (s *Service) NotificationHistory(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	return s.store.NotificationHistory(ctx, userID)
}

func (s *Service) AuditHistory(ctx context.Context, userID uuid.UUID) ([]domain.AuditEvent, error) {
	return s.store.AuditHistory(ctx, userID)
}

func (s *Service) ProcessLifecycleTick(ctx context.Context) error {
	events, err := s.store.TransitionOverdueWills(ctx, s.now().UTC())
	if err != nil {
		return err
	}
	var notifications []domain.Notification
	now := s.now().UTC()
	for _, event := range events {
		switch event.CurrentState {
		case domain.LifecyclePendingVerification:
			notifications = append(notifications, dormancyNotifications(event, now)...)
			notifications = append(notifications, verificationStartNotifications(event, now)...)
		case domain.LifecycleReadyForExecution:
		}
	}
	if len(notifications) == 0 {
		return nil
	}
	if err = s.queue.Enqueue(ctx, notifications); err != nil {
		return err
	}
	return s.deliverPending(ctx, 50)
}

func (s *Service) deliverPending(ctx context.Context, limit int) error {
	if s.queue == nil || s.notifier == nil {
		return nil
	}
	notifications, err := s.queue.Pending(ctx, limit)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for _, notification := range notifications {
		if err = s.notifier.Send(ctx, notification); err != nil {
			if markErr := s.queue.MarkFailed(ctx, notification.ID, err.Error(), now); markErr != nil {
				return markErr
			}
			continue
		}
		if err = s.queue.MarkSent(ctx, notification.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) respondVerification(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID, decision domain.VerificationDecision) error {
	result, err := s.store.RecordVerificationDecision(ctx, strings.ToLower(strings.TrimSpace(trusteeEmail)), actorUserID, requestID, decision, s.now().UTC())
	if err != nil {
		return err
	}
	if result.TransitionedTo == nil || *result.TransitionedTo != domain.LifecycleGracePeriod {
		return nil
	}
	if err = s.queue.Enqueue(ctx, graceNotifications(result, s.now().UTC())); err != nil {
		return err
	}
	return s.deliverPending(ctx, 20)
}

func normalizeWillInput(input domain.UpsertWillInput) (domain.UpsertWillInput, error) {
	normalized := domain.UpsertWillInput{
		Status:                input.Status,
		DormancyPeriodDays:    input.DormancyPeriodDays,
		GracePeriodDays:       input.GracePeriodDays,
		PolicyVersionAccepted: strings.TrimSpace(input.PolicyVersionAccepted),
	}

	if input.Status != domain.StatusDraft && input.Status != domain.StatusPublished {
		return domain.UpsertWillInput{}, ValidationError{message: "state must be either draft or published"}
	}
	if input.DormancyPeriodDays < 1 || input.DormancyPeriodDays > 3650 {
		return domain.UpsertWillInput{}, ValidationError{message: "dormancyPeriodDays must be between 1 and 3650"}
	}
	if input.GracePeriodDays < 1 || input.GracePeriodDays > 365 {
		return domain.UpsertWillInput{}, ValidationError{message: "gracePeriodDays must be between 1 and 365"}
	}
	if normalized.PolicyVersionAccepted == "" || len(normalized.PolicyVersionAccepted) > 64 {
		return domain.UpsertWillInput{}, ValidationError{message: "policyVersionAccepted is required and must not exceed 64 characters"}
	}

	categories := map[domain.ReleaseCategory]struct{}{}
	for _, category := range input.ReleaseCategories {
		value := domain.ReleaseCategory(strings.TrimSpace(string(category)))
		if _, ok := allowedCategories[value]; !ok {
			return domain.UpsertWillInput{}, ValidationError{message: "releaseCategories contain an unsupported value"}
		}
		categories[value] = struct{}{}
	}
	if len(categories) == 0 {
		return domain.UpsertWillInput{}, ValidationError{message: "at least one release category is required"}
	}

	normalized.ReleaseCategories = make([]domain.ReleaseCategory, 0, len(categories))
	for category := range categories {
		normalized.ReleaseCategories = append(normalized.ReleaseCategories, category)
	}
	slices.Sort(normalized.ReleaseCategories)
	return normalized, nil
}

func normalizeTrusteeInput(input domain.TrusteeInput) (domain.TrusteeInput, error) {
	normalized := domain.TrusteeInput{
		Name:         strings.TrimSpace(input.Name),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		Relationship: strings.TrimSpace(input.Relationship),
	}
	if normalized.Name == "" || len(normalized.Name) > 100 {
		return domain.TrusteeInput{}, ValidationError{message: "name is required and must not exceed 100 characters"}
	}
	address, err := mail.ParseAddress(normalized.Email)
	if err != nil || address.Address != normalized.Email || len(normalized.Email) > 254 {
		return domain.TrusteeInput{}, ValidationError{message: "email must be valid"}
	}
	if normalized.Relationship == "" || len(normalized.Relationship) > 100 {
		return domain.TrusteeInput{}, ValidationError{message: "relationship is required and must not exceed 100 characters"}
	}
	return normalized, nil
}

func dormancyNotifications(event LifecycleEvent, now time.Time) []domain.Notification {
	return []domain.Notification{{
		ID:             uuid.New(),
		WillID:         event.WillID,
		UserID:         event.UserID,
		EventType:      "user_became_dormant",
		Channel:        "email",
		RecipientName:  displayName(event.OwnerDisplayName, event.OwnerEmail),
		RecipientEmail: event.OwnerEmail,
		Subject:        "Waaris dormancy detected",
		Body:           "Your Digital Will has entered pending verification because no heartbeat was received within the configured dormancy period.",
		Status:         "queued",
		QueuedAt:       now,
	}}
}

func verificationStartNotifications(event LifecycleEvent, now time.Time) []domain.Notification {
	notifications := make([]domain.Notification, 0, len(event.Trustees))
	for _, trustee := range event.Trustees {
		trusteeID := trustee.ID
		notifications = append(notifications, domain.Notification{
			ID:             uuid.New(),
			WillID:         event.WillID,
			UserID:         event.UserID,
			TrusteeID:      &trusteeID,
			EventType:      "verification_begins",
			Channel:        "email",
			RecipientName:  trustee.Name,
			RecipientEmail: trustee.Email,
			Subject:        "Waaris verification requested",
			Body:           fmt.Sprintf("Verification has started for %s. Please review the pending verification request in Waaris.", displayName(event.OwnerDisplayName, event.OwnerEmail)),
			Status:         "queued",
			QueuedAt:       now,
		})
	}
	return notifications
}

func graceNotifications(result VerificationDecisionResult, now time.Time) []domain.Notification {
	notifications := []domain.Notification{{
		ID:             uuid.New(),
		WillID:         result.WillID,
		UserID:         result.UserID,
		EventType:      "grace_period_begins",
		Channel:        "email",
		RecipientName:  displayName(result.OwnerDisplayName, result.OwnerEmail),
		RecipientEmail: result.OwnerEmail,
		Subject:        "Waaris grace period has started",
		Body:           "Your Digital Will has entered the grace period after trustee verification. A fresh heartbeat will restore your status to active.",
		Status:         "queued",
		QueuedAt:       now,
	}}
	for _, trustee := range result.Trustees {
		trusteeID := trustee.ID
		notifications = append(notifications, domain.Notification{
			ID:             uuid.New(),
			WillID:         result.WillID,
			UserID:         result.UserID,
			TrusteeID:      &trusteeID,
			EventType:      "grace_period_begins",
			Channel:        "email",
			RecipientName:  trustee.Name,
			RecipientEmail: trustee.Email,
			Subject:        "Waaris grace period has started",
			Body:           fmt.Sprintf("%s has entered the Waaris grace period after trustee verification.", displayName(result.OwnerDisplayName, result.OwnerEmail)),
			Status:         "queued",
			QueuedAt:       now,
		})
	}
	return notifications
}

func recoveryNotifications(will domain.DigitalWill, trustees []domain.Trustee, now time.Time) []domain.Notification {
	notifications := []domain.Notification{{
		ID:             uuid.New(),
		WillID:         will.ID,
		UserID:         will.UserID,
		EventType:      "liveness_restored",
		Channel:        "email",
		RecipientName:  displayName(will.OwnerDisplayName, will.OwnerEmail),
		RecipientEmail: will.OwnerEmail,
		Subject:        "Waaris liveness restored",
		Body:           "A heartbeat restored your Digital Will lifecycle state to active.",
		Status:         "queued",
		QueuedAt:       now,
	}}
	for _, trustee := range trustees {
		trusteeID := trustee.ID
		notifications = append(notifications, domain.Notification{
			ID:             uuid.New(),
			WillID:         will.ID,
			UserID:         will.UserID,
			TrusteeID:      &trusteeID,
			EventType:      "liveness_restored",
			Channel:        "email",
			RecipientName:  trustee.Name,
			RecipientEmail: trustee.Email,
			Subject:        "Waaris liveness restored",
			Body:           fmt.Sprintf("%s restored liveness and the verification workflow has been reset.", displayName(will.OwnerDisplayName, will.OwnerEmail)),
			Status:         "queued",
			QueuedAt:       now,
		})
	}
	return notifications
}

func displayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return email
}

func isValidationError(err error) bool {
	var validationError ValidationError
	return errors.As(err, &validationError)
}
