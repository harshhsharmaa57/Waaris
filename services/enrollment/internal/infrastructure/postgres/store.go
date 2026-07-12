package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/enrollment/internal/application"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	defer tx.Rollback(ctx)

	willID := uuid.New()
	versionID := uuid.New()
	consentID := uuid.New()

	var will domain.DigitalWill
	err = tx.QueryRow(ctx, `
		INSERT INTO waaris.digital_wills (
			id, user_id, status, lifecycle_state, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $8, $8)
		RETURNING id, user_id, status, lifecycle_state, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, last_heartbeat_at, pending_verification_started_at, grace_period_started_at, ready_for_execution_at, created_at, updated_at
	`, willID, userID, input.Status, domain.LifecycleActive, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now).Scan(
		&will.ID, &will.UserID, &will.Status, &will.LifecycleState, &will.Version, &will.DormancyPeriodDays, &will.GracePeriodDays, &will.PolicyVersionAccepted, &will.ConsentAcceptedAt, &will.LastHeartbeatAt, &will.PendingVerificationStartedAt, &will.GracePeriodStartedAt, &will.ReadyForExecutionAt, &will.CreatedAt, &will.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return domain.DigitalWill{}, domain.ErrAlreadyExists
	}
	if err != nil {
		return domain.DigitalWill{}, err
	}
	if err = s.populateOwner(ctx, tx, &will); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = insertReleasePreferences(ctx, tx, "waaris.will_release_preferences", "will_id", willID, input.ReleaseCategories, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.will_versions (id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at)
		VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8, $8)
	`, versionID, willID, userID, input.Status, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = insertReleasePreferences(ctx, tx, "waaris.will_version_release_preferences", "will_version_id", versionID, input.ReleaseCategories, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.consent_records (id, will_id, will_version_id, user_id, policy_version, consent_type, accepted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, consentID, willID, versionID, userID, input.PolicyVersionAccepted, domain.ConsentTypeTerms, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "will_created", "{}", now); err != nil {
		return domain.DigitalWill{}, err
	}
	will.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	if err = tx.Commit(ctx); err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) WillByUser(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) UpdateWill(ctx context.Context, userID uuid.UUID, input domain.UpsertWillInput, now time.Time) (domain.DigitalWill, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	defer tx.Rollback(ctx)

	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	nextVersion := will.Version + 1
	versionID := uuid.New()
	consentID := uuid.New()

	err = tx.QueryRow(ctx, `
		UPDATE waaris.digital_wills
		SET status = $2, current_version = $3, dormancy_days = $4, grace_days = $5, policy_version = $6, consent_accepted_at = $7, updated_at = $7
		WHERE id = $1
		RETURNING id, user_id, status, lifecycle_state, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, last_heartbeat_at, pending_verification_started_at, grace_period_started_at, ready_for_execution_at, created_at, updated_at
	`, will.ID, input.Status, nextVersion, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now).Scan(
		&will.ID, &will.UserID, &will.Status, &will.LifecycleState, &will.Version, &will.DormancyPeriodDays, &will.GracePeriodDays, &will.PolicyVersionAccepted, &will.ConsentAcceptedAt, &will.LastHeartbeatAt, &will.PendingVerificationStartedAt, &will.GracePeriodStartedAt, &will.ReadyForExecutionAt, &will.CreatedAt, &will.UpdatedAt,
	)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM waaris.will_release_preferences WHERE will_id = $1`, will.ID); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = insertReleasePreferences(ctx, tx, "waaris.will_release_preferences", "will_id", will.ID, input.ReleaseCategories, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.will_versions (id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, versionID, will.ID, userID, nextVersion, input.Status, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = insertReleasePreferences(ctx, tx, "waaris.will_version_release_preferences", "will_version_id", versionID, input.ReleaseCategories, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.consent_records (id, will_id, will_version_id, user_id, policy_version, consent_type, accepted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, consentID, will.ID, versionID, userID, input.PolicyVersionAccepted, domain.ConsentTypeTerms, now); err != nil {
		return domain.DigitalWill{}, err
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "will_updated", "{}", now); err != nil {
		return domain.DigitalWill{}, err
	}
	will.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	if err = tx.Commit(ctx); err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) DeleteWill(ctx context.Context, userID uuid.UUID, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE waaris.digital_wills SET deleted_at = $2, updated_at = $2 WHERE user_id = $1 AND deleted_at IS NULL`, userID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "will_deleted", "{}", now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) WillHistory(ctx context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at
		FROM waaris.will_versions WHERE will_id = $1 ORDER BY version DESC
	`, will.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []domain.WillVersion
	for rows.Next() {
		var item domain.WillVersion
		if err = rows.Scan(&item.ID, &item.WillID, &item.UserID, &item.Version, &item.Status, &item.DormancyPeriodDays, &item.GracePeriodDays, &item.PolicyVersionAccepted, &item.ConsentAcceptedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ReleaseCategories, err = loadCategories(ctx, s.pool, `SELECT category FROM waaris.will_version_release_preferences WHERE will_version_id = $1 ORDER BY category`, item.ID)
		if err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}

func (s *Store) TrusteeCount(ctx context.Context, userID uuid.UUID) (int, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM waaris.trustees WHERE will_id = $1`, will.ID).Scan(&count)
	return count, err
}

func (s *Store) CreateTrustee(ctx context.Context, userID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Trustee{}, err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return domain.Trustee{}, err
	}
	trustee := domain.Trustee{ID: uuid.New(), WillID: will.ID, UserID: userID}
	err = tx.QueryRow(ctx, `
		INSERT INTO waaris.trustees (id, will_id, user_id, name, email, relationship, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING name, email, relationship, created_at, updated_at
	`, trustee.ID, will.ID, userID, input.Name, input.Email, input.Relationship, now).Scan(&trustee.Name, &trustee.Email, &trustee.Relationship, &trustee.CreatedAt, &trustee.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.Trustee{}, domain.ErrDuplicateTrustee
	}
	if err != nil {
		return domain.Trustee{}, err
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "trustee_created", `{"trusteeId":"`+trustee.ID.String()+`"}`, now); err != nil {
		return domain.Trustee{}, err
	}
	return trustee, tx.Commit(ctx)
}

func (s *Store) TrusteesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Trustee, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return nil, err
	}
	return s.trusteesByWill(ctx, s.pool, will.ID)
}

func (s *Store) UpdateTrustee(ctx context.Context, userID uuid.UUID, trusteeID uuid.UUID, input domain.TrusteeInput, now time.Time) (domain.Trustee, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Trustee{}, err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return domain.Trustee{}, err
	}
	trustee := domain.Trustee{}
	err = tx.QueryRow(ctx, `
		UPDATE waaris.trustees
		SET name = $3, email = $4, relationship = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
		RETURNING id, will_id, user_id, name, email, relationship, created_at, updated_at
	`, trusteeID, userID, input.Name, input.Email, input.Relationship, now).Scan(&trustee.ID, &trustee.WillID, &trustee.UserID, &trustee.Name, &trustee.Email, &trustee.Relationship, &trustee.CreatedAt, &trustee.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.Trustee{}, domain.ErrDuplicateTrustee
	}
	if err != nil {
		return domain.Trustee{}, mapNotFound(err)
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "trustee_updated", `{"trusteeId":"`+trustee.ID.String()+`"}`, now); err != nil {
		return domain.Trustee{}, err
	}
	return trustee, tx.Commit(ctx)
}

func (s *Store) DeleteTrustee(ctx context.Context, userID uuid.UUID, trusteeID uuid.UUID, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `DELETE FROM waaris.trustees WHERE id = $1 AND user_id = $2`, trusteeID, userID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "trustee_deleted", `{"trusteeId":"`+trusteeID.String()+`"}`, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SubmitHeartbeat(ctx context.Context, userID uuid.UUID, source string, now time.Time) (domain.HeartbeatStatus, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.HeartbeatStatus{}, false, err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return domain.HeartbeatStatus{}, false, err
	}
	heartbeatID := uuid.New()
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.heartbeats (id, will_id, user_id, source, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, heartbeatID, will.ID, userID, source, now); err != nil {
		return domain.HeartbeatStatus{}, false, err
	}
	restored := will.LifecycleState != domain.LifecycleActive
	if restored {
		if _, err = tx.Exec(ctx, `
			UPDATE waaris.digital_wills
			SET lifecycle_state = $2, last_heartbeat_at = $3, pending_verification_started_at = NULL, grace_period_started_at = NULL, ready_for_execution_at = NULL, updated_at = $3
			WHERE id = $1
		`, will.ID, domain.LifecycleActive, now); err != nil {
			return domain.HeartbeatStatus{}, false, err
		}
		if _, err = tx.Exec(ctx, `UPDATE waaris.verification_requests SET status = 'cancelled', resolved_at = $2 WHERE will_id = $1 AND status = 'pending'`, will.ID, now); err != nil {
			return domain.HeartbeatStatus{}, false, err
		}
		if err = appendAudit(ctx, tx, will, "user", userID.String(), "liveness_restored", "{}", now); err != nil {
			return domain.HeartbeatStatus{}, false, err
		}
		will.LifecycleState = domain.LifecycleActive
		will.PendingVerificationStartedAt = nil
		will.GracePeriodStartedAt = nil
		will.ReadyForExecutionAt = nil
	} else {
		if _, err = tx.Exec(ctx, `UPDATE waaris.digital_wills SET last_heartbeat_at = $2, updated_at = $2 WHERE id = $1`, will.ID, now); err != nil {
			return domain.HeartbeatStatus{}, false, err
		}
	}
	will.LastHeartbeatAt = ptrTime(now)
	if err = appendAudit(ctx, tx, will, "user", userID.String(), "heartbeat_submitted", "{}", now); err != nil {
		return domain.HeartbeatStatus{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.HeartbeatStatus{}, false, err
	}
	return heartbeatStatusFromWill(will), restored, nil
}

func (s *Store) HeartbeatStatus(ctx context.Context, userID uuid.UUID) (domain.HeartbeatStatus, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return domain.HeartbeatStatus{}, err
	}
	return heartbeatStatusFromWill(will), nil
}

func (s *Store) HeartbeatHistory(ctx context.Context, userID uuid.UUID) ([]domain.Heartbeat, error) {
	will, err := s.loadWill(ctx, s.pool, userID, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, will_id, user_id, source, occurred_at, created_at FROM waaris.heartbeats WHERE will_id = $1 ORDER BY occurred_at DESC`, will.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []domain.Heartbeat
	for rows.Next() {
		var item domain.Heartbeat
		if err = rows.Scan(&item.ID, &item.WillID, &item.UserID, &item.Source, &item.OccurredAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}

func (s *Store) TransitionOverdueWills(ctx context.Context, now time.Time) ([]application.LifecycleEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id
		FROM waaris.digital_wills
		WHERE deleted_at IS NULL
		  AND status = 'published'
		  AND lifecycle_state IN ('active','grace_period')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err = rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	var events []application.LifecycleEvent
	for _, userID := range userIDs {
		event, ok, err := s.transitionSingleWill(ctx, userID, now)
		if err != nil {
			return nil, err
		}
		if ok {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *Store) PendingVerifications(ctx context.Context, trusteeEmail string) ([]domain.VerificationRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vr.id, vr.will_id, vr.user_id, vr.threshold_required, vr.status, vr.created_at, vr.resolved_at,
		       u.email, u.display_name,
		       t.id, t.will_id, t.user_id, t.name, t.email, t.relationship, t.created_at, t.updated_at,
		       latest.decision
		FROM waaris.verification_requests vr
		JOIN waaris.users u ON u.id = vr.user_id
		JOIN waaris.trustees t ON t.will_id = vr.will_id
		LEFT JOIN LATERAL (
			SELECT decision
			FROM waaris.verification_responses
			WHERE request_id = vr.id AND trustee_id = t.id
			ORDER BY responded_at DESC
			LIMIT 1
		) latest ON true
		WHERE vr.status = 'pending' AND lower(t.email) = $1
		ORDER BY vr.created_at DESC
	`, strings.ToLower(trusteeEmail))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.VerificationRequest
	for rows.Next() {
		var item domain.VerificationRequest
		var latestDecision *string
		if err = rows.Scan(
			&item.ID, &item.WillID, &item.UserID, &item.ThresholdRequired, &item.Status, &item.CreatedAt, &item.ResolvedAt,
			&item.OwnerEmail, &item.OwnerDisplayName,
			&item.Trustee.ID, &item.Trustee.WillID, &item.Trustee.UserID, &item.Trustee.Name, &item.Trustee.Email, &item.Trustee.Relationship, &item.Trustee.CreatedAt, &item.Trustee.UpdatedAt,
			&latestDecision,
		); err != nil {
			return nil, err
		}
		if latestDecision != nil {
			decision := domain.VerificationDecision(*latestDecision)
			item.LatestDecision = &decision
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) RecordVerificationDecision(ctx context.Context, trusteeEmail string, actorUserID, requestID uuid.UUID, decision domain.VerificationDecision, now time.Time) (application.VerificationDecisionResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return application.VerificationDecisionResult{}, err
	}
	defer tx.Rollback(ctx)

	var result application.VerificationDecisionResult
	var ownerWill domain.DigitalWill
	err = tx.QueryRow(ctx, `
		SELECT vr.will_id, vr.user_id, vr.threshold_required, vr.status,
		       dw.id, dw.user_id, dw.status, dw.lifecycle_state, dw.current_version, dw.dormancy_days, dw.grace_days, dw.policy_version, dw.consent_accepted_at, dw.last_heartbeat_at, dw.pending_verification_started_at, dw.grace_period_started_at, dw.ready_for_execution_at, dw.created_at, dw.updated_at,
		       u.email, u.display_name
		FROM waaris.verification_requests vr
		JOIN waaris.digital_wills dw ON dw.id = vr.will_id
		JOIN waaris.users u ON u.id = vr.user_id
		WHERE vr.id = $1
		FOR UPDATE
	`, requestID).Scan(
		&result.WillID, &result.UserID, &result.ThresholdRequired, new(string),
		&ownerWill.ID, &ownerWill.UserID, &ownerWill.Status, &ownerWill.LifecycleState, &ownerWill.Version, &ownerWill.DormancyPeriodDays, &ownerWill.GracePeriodDays, &ownerWill.PolicyVersionAccepted, &ownerWill.ConsentAcceptedAt, &ownerWill.LastHeartbeatAt, &ownerWill.PendingVerificationStartedAt, &ownerWill.GracePeriodStartedAt, &ownerWill.ReadyForExecutionAt, &ownerWill.CreatedAt, &ownerWill.UpdatedAt,
		&ownerWill.OwnerEmail, &ownerWill.OwnerDisplayName,
	)
	if err != nil {
		return application.VerificationDecisionResult{}, mapNotFound(err)
	}
	if ownerWill.LifecycleState != domain.LifecyclePendingVerification {
		return application.VerificationDecisionResult{}, domain.ErrVerificationNotPending
	}
	var requestStatus string
	if err = tx.QueryRow(ctx, `SELECT status FROM waaris.verification_requests WHERE id = $1`, requestID).Scan(&requestStatus); err != nil {
		return application.VerificationDecisionResult{}, err
	}
	if requestStatus != "pending" {
		return application.VerificationDecisionResult{}, domain.ErrVerificationNotPending
	}
	var trustee domain.Trustee
	err = tx.QueryRow(ctx, `
		SELECT id, will_id, user_id, name, email, relationship, created_at, updated_at
		FROM waaris.trustees
		WHERE will_id = $1 AND lower(email) = $2
	`, result.WillID, strings.ToLower(trusteeEmail)).Scan(&trustee.ID, &trustee.WillID, &trustee.UserID, &trustee.Name, &trustee.Email, &trustee.Relationship, &trustee.CreatedAt, &trustee.UpdatedAt)
	if err != nil {
		return application.VerificationDecisionResult{}, domain.ErrTrusteeNotAssigned
	}
	responseID := uuid.New()
	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.verification_responses (id, request_id, will_id, trustee_id, actor_user_id, decision, responded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, responseID, requestID, result.WillID, trustee.ID, actorUserID, decision, now); err != nil {
		return application.VerificationDecisionResult{}, err
	}
	result.ActorTrustee = trustee
	result.OwnerEmail = ownerWill.OwnerEmail
	result.OwnerDisplayName = ownerWill.OwnerDisplayName
	result.RequestID = requestID
	result.Decision = decision
	result.GracePeriodDays = ownerWill.GracePeriodDays
	result.Trustees, err = s.trusteesByWill(ctx, tx, result.WillID)
	if err != nil {
		return application.VerificationDecisionResult{}, err
	}
	result.Approvals, err = approvalCount(ctx, tx, requestID)
	if err != nil {
		return application.VerificationDecisionResult{}, err
	}
	if err = appendAudit(ctx, tx, ownerWill, "trustee", trustee.Email, "trustee_response_recorded", `{"requestId":"`+requestID.String()+`","decision":"`+string(decision)+`"}`, now); err != nil {
		return application.VerificationDecisionResult{}, err
	}
	if result.Approvals >= result.ThresholdRequired {
		if _, err = tx.Exec(ctx, `
			UPDATE waaris.digital_wills
			SET lifecycle_state = $2, grace_period_started_at = $3, updated_at = $3
			WHERE id = $1
		`, ownerWill.ID, domain.LifecycleGracePeriod, now); err != nil {
			return application.VerificationDecisionResult{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE waaris.verification_requests SET status = 'resolved', resolved_at = $2 WHERE id = $1`, requestID, now); err != nil {
			return application.VerificationDecisionResult{}, err
		}
		state := domain.LifecycleGracePeriod
		result.TransitionedTo = &state
		if err = appendAudit(ctx, tx, ownerWill, "system", "verification-threshold", "grace_period_started", `{"requestId":"`+requestID.String()+`"}`, now); err != nil {
			return application.VerificationDecisionResult{}, err
		}
	}
	return result, tx.Commit(ctx)
}

func (s *Store) NotificationHistory(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, will_id, user_id, trustee_id, event_type, channel, recipient_name, recipient_email, subject, body, status, queued_at, sent_at, COALESCE(failure_message, '')
		FROM waaris.notifications WHERE user_id = $1 ORDER BY queued_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Notification
	for rows.Next() {
		var item domain.Notification
		if err = rows.Scan(&item.ID, &item.WillID, &item.UserID, &item.TrusteeID, &item.EventType, &item.Channel, &item.RecipientName, &item.RecipientEmail, &item.Subject, &item.Body, &item.Status, &item.QueuedAt, &item.SentAt, &item.FailureMessage); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) AuditHistory(ctx context.Context, userID uuid.UUID) ([]domain.AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, will_id, actor_type, actor_id, event_type, correlation_id, details::text, occurred_at
		FROM waaris.audit_events WHERE user_id = $1 ORDER BY occurred_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.AuditEvent
	for rows.Next() {
		var item domain.AuditEvent
		if err = rows.Scan(&item.ID, &item.UserID, &item.WillID, &item.ActorType, &item.ActorID, &item.EventType, &item.CorrelationID, &item.Details, &item.OccurredAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) Enqueue(ctx context.Context, notifications []domain.Notification) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, notification := range notifications {
		if _, err = tx.Exec(ctx, `
			INSERT INTO waaris.notifications (id, will_id, user_id, trustee_id, event_type, channel, recipient_name, recipient_email, subject, body, status, queued_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, notification.ID, notification.WillID, notification.UserID, notification.TrusteeID, notification.EventType, notification.Channel, notification.RecipientName, notification.RecipientEmail, notification.Subject, notification.Body, notification.Status, notification.QueuedAt); err != nil {
			return err
		}
		will, err := s.loadWill(ctx, tx, notification.UserID, false)
		if err == nil {
			if err = appendAudit(ctx, tx, will, "system", "notification-queue", "notification_queued", `{"notificationId":"`+notification.ID.String()+`","eventType":"`+notification.EventType+`"}`, notification.QueuedAt); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Pending(ctx context.Context, limit int) ([]domain.Notification, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, will_id, user_id, trustee_id, event_type, channel, recipient_name, recipient_email, subject, body, status, queued_at, sent_at, COALESCE(failure_message, '')
		FROM waaris.notifications WHERE status = 'queued' ORDER BY queued_at ASC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Notification
	for rows.Next() {
		var item domain.Notification
		if err = rows.Scan(&item.ID, &item.WillID, &item.UserID, &item.TrusteeID, &item.EventType, &item.Channel, &item.RecipientName, &item.RecipientEmail, &item.Subject, &item.Body, &item.Status, &item.QueuedAt, &item.SentAt, &item.FailureMessage); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) MarkSent(ctx context.Context, notificationID uuid.UUID, sentAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var userID uuid.UUID
	command, err := tx.Exec(ctx, `UPDATE waaris.notifications SET status = 'sent', sent_at = $2 WHERE id = $1`, notificationID, sentAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err = tx.QueryRow(ctx, `SELECT user_id FROM waaris.notifications WHERE id = $1`, notificationID).Scan(&userID); err == nil {
		if will, loadErr := s.loadWill(ctx, tx, userID, false); loadErr == nil {
			if err = appendAudit(ctx, tx, will, "system", "notification-dispatch", "notification_sent", `{"notificationId":"`+notificationID.String()+`"}`, sentAt); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkFailed(ctx context.Context, notificationID uuid.UUID, message string, failedAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var userID uuid.UUID
	command, err := tx.Exec(ctx, `UPDATE waaris.notifications SET status = 'failed', failure_message = $2 WHERE id = $1`, notificationID, message)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err = tx.QueryRow(ctx, `SELECT user_id FROM waaris.notifications WHERE id = $1`, notificationID).Scan(&userID); err == nil {
		if will, loadErr := s.loadWill(ctx, tx, userID, false); loadErr == nil {
			if err = appendAudit(ctx, tx, will, "system", "notification-dispatch", "notification_failed", `{"notificationId":"`+notificationID.String()+`"}`, failedAt); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

type executor interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *Store) loadWill(ctx context.Context, q executor, userID uuid.UUID, forUpdate bool) (domain.DigitalWill, error) {
	query := `
		SELECT dw.id, dw.user_id, u.email, u.display_name, dw.status, dw.lifecycle_state, dw.current_version, dw.dormancy_days, dw.grace_days, dw.policy_version, dw.consent_accepted_at, dw.last_heartbeat_at, dw.pending_verification_started_at, dw.grace_period_started_at, dw.ready_for_execution_at, dw.created_at, dw.updated_at
		FROM waaris.digital_wills dw
		JOIN waaris.users u ON u.id = dw.user_id
		WHERE dw.user_id = $1 AND dw.deleted_at IS NULL
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var will domain.DigitalWill
	err := q.QueryRow(ctx, query, userID).Scan(
		&will.ID, &will.UserID, &will.OwnerEmail, &will.OwnerDisplayName, &will.Status, &will.LifecycleState, &will.Version, &will.DormancyPeriodDays, &will.GracePeriodDays, &will.PolicyVersionAccepted, &will.ConsentAcceptedAt, &will.LastHeartbeatAt, &will.PendingVerificationStartedAt, &will.GracePeriodStartedAt, &will.ReadyForExecutionAt, &will.CreatedAt, &will.UpdatedAt,
	)
	if err != nil {
		return domain.DigitalWill{}, mapNotFound(err)
	}
	will.ReleaseCategories, err = loadCategories(ctx, q, `SELECT category FROM waaris.will_release_preferences WHERE will_id = $1 ORDER BY category`, will.ID)
	if err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) populateOwner(ctx context.Context, q executor, will *domain.DigitalWill) error {
	return q.QueryRow(ctx, `SELECT email, display_name FROM waaris.users WHERE id = $1`, will.UserID).Scan(&will.OwnerEmail, &will.OwnerDisplayName)
}

func (s *Store) trusteesByWill(ctx context.Context, q executor, willID uuid.UUID) ([]domain.Trustee, error) {
	rows, err := q.Query(ctx, `
		SELECT id, will_id, user_id, name, email, relationship, created_at, updated_at
		FROM waaris.trustees WHERE will_id = $1 ORDER BY email ASC
	`, willID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trustees []domain.Trustee
	for rows.Next() {
		var trustee domain.Trustee
		if err = rows.Scan(&trustee.ID, &trustee.WillID, &trustee.UserID, &trustee.Name, &trustee.Email, &trustee.Relationship, &trustee.CreatedAt, &trustee.UpdatedAt); err != nil {
			return nil, err
		}
		trustees = append(trustees, trustee)
	}
	return trustees, rows.Err()
}

func (s *Store) transitionSingleWill(ctx context.Context, userID uuid.UUID, now time.Time) (application.LifecycleEvent, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return application.LifecycleEvent{}, false, err
	}
	defer tx.Rollback(ctx)
	will, err := s.loadWill(ctx, tx, userID, true)
	if err != nil {
		return application.LifecycleEvent{}, false, err
	}
	trustees, err := s.trusteesByWill(ctx, tx, will.ID)
	if err != nil {
		return application.LifecycleEvent{}, false, err
	}
	switch will.LifecycleState {
	case domain.LifecycleActive:
		reference := will.UpdatedAt
		if will.LastHeartbeatAt != nil {
			reference = *will.LastHeartbeatAt
		}
		if now.Before(reference.Add(time.Duration(will.DormancyPeriodDays)*24*time.Hour)) || len(trustees) == 0 {
			return application.LifecycleEvent{}, false, nil
		}
		threshold := (len(trustees) / 2) + 1
		requestID := uuid.New()
		if _, err = tx.Exec(ctx, `
			UPDATE waaris.digital_wills
			SET lifecycle_state = $2, pending_verification_started_at = $3, updated_at = $3
			WHERE id = $1
		`, will.ID, domain.LifecyclePendingVerification, now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO waaris.verification_requests (id, will_id, user_id, threshold_required, status, created_at)
			VALUES ($1, $2, $3, $4, 'pending', $5)
		`, requestID, will.ID, userID, threshold, now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		if err = appendAudit(ctx, tx, will, "system", "lifecycle-tick", "dormancy_detected", "{}", now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		if err = appendAudit(ctx, tx, will, "system", "lifecycle-tick", "verification_started", `{"requestId":"`+requestID.String()+`"}`, now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		event := application.LifecycleEvent{UserID: userID, WillID: will.ID, OwnerEmail: will.OwnerEmail, OwnerDisplayName: will.OwnerDisplayName, ThresholdRequired: threshold, Trustees: trustees, TransitionedAt: now, PreviousState: domain.LifecycleActive, CurrentState: domain.LifecyclePendingVerification, GracePeriodDays: will.GracePeriodDays, VerificationRequest: &requestID}
		return event, true, tx.Commit(ctx)
	case domain.LifecycleGracePeriod:
		if will.GracePeriodStartedAt == nil || now.Before(will.GracePeriodStartedAt.Add(time.Duration(will.GracePeriodDays)*24*time.Hour)) {
			return application.LifecycleEvent{}, false, nil
		}
		if _, err = tx.Exec(ctx, `
			UPDATE waaris.digital_wills
			SET lifecycle_state = $2, ready_for_execution_at = $3, updated_at = $3
			WHERE id = $1
		`, will.ID, domain.LifecycleReadyForExecution, now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		if err = appendAudit(ctx, tx, will, "system", "lifecycle-tick", "ready_for_execution", "{}", now); err != nil {
			return application.LifecycleEvent{}, false, err
		}
		event := application.LifecycleEvent{UserID: userID, WillID: will.ID, OwnerEmail: will.OwnerEmail, OwnerDisplayName: will.OwnerDisplayName, Trustees: trustees, TransitionedAt: now, PreviousState: domain.LifecycleGracePeriod, CurrentState: domain.LifecycleReadyForExecution}
		return event, true, tx.Commit(ctx)
	default:
		return application.LifecycleEvent{}, false, nil
	}
}

func approvalCount(ctx context.Context, q executor, requestID uuid.UUID) (int, error) {
	var approvals int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT DISTINCT ON (trustee_id) trustee_id, decision
			FROM waaris.verification_responses
			WHERE request_id = $1
			ORDER BY trustee_id, responded_at DESC
		) latest
		WHERE decision = 'approve'
	`, requestID).Scan(&approvals)
	return approvals, err
}

func appendAudit(ctx context.Context, q executor, will domain.DigitalWill, actorType, actorID, eventType, details string, now time.Time) error {
	correlationID := application.CorrelationID(ctx)
	if _, err := q.Exec(ctx, `
		INSERT INTO waaris.audit_events (id, user_id, will_id, actor_type, actor_id, event_type, correlation_id, details, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
	`, uuid.New(), will.UserID, will.ID, actorType, actorID, eventType, correlationID, details, now); err != nil {
		return err
	}
	return nil
}

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertReleasePreferences(ctx context.Context, q queryer, table, keyColumn string, key uuid.UUID, categories []domain.ReleaseCategory, now time.Time) error {
	for _, category := range categories {
		if _, err := q.Exec(ctx, `INSERT INTO `+table+` (`+keyColumn+`, category, created_at) VALUES ($1, $2, $3)`, key, category, now); err != nil {
			return err
		}
	}
	return nil
}

func loadCategories(ctx context.Context, q queryer, query string, key uuid.UUID) ([]domain.ReleaseCategory, error) {
	rows, err := q.Query(ctx, query, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []domain.ReleaseCategory
	for rows.Next() {
		var category domain.ReleaseCategory
		if err = rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func heartbeatStatusFromWill(will domain.DigitalWill) domain.HeartbeatStatus {
	return domain.HeartbeatStatus{
		WillID:                       will.ID,
		LifecycleState:               will.LifecycleState,
		LastHeartbeatAt:              will.LastHeartbeatAt,
		PendingVerificationStartedAt: will.PendingVerificationStartedAt,
		GracePeriodStartedAt:         will.GracePeriodStartedAt,
		ReadyForExecutionAt:          will.ReadyForExecutionAt,
	}
}

func ptrTime(value time.Time) *time.Time {
	result := value.UTC()
	return &result
}
