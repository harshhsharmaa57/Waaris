package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

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
			id, user_id, status, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
		) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $7, $7)
		RETURNING id, user_id, status, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
	`, willID, userID, input.Status, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now).Scan(
		&will.ID,
		&will.UserID,
		&will.Status,
		&will.Version,
		&will.DormancyPeriodDays,
		&will.GracePeriodDays,
		&will.PolicyVersionAccepted,
		&will.ConsentAcceptedAt,
		&will.CreatedAt,
		&will.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return domain.DigitalWill{}, domain.ErrAlreadyExists
	}
	if err != nil {
		return domain.DigitalWill{}, err
	}

	if err = insertReleasePreferences(ctx, tx, "waaris.will_release_preferences", "will_id", willID, input.ReleaseCategories, now); err != nil {
		return domain.DigitalWill{}, err
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO waaris.will_versions (
			id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at
		) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8, $8)
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

	will.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	if err = tx.Commit(ctx); err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) WillByUser(ctx context.Context, userID uuid.UUID) (domain.DigitalWill, error) {
	var will domain.DigitalWill
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, status, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
		FROM waaris.digital_wills
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID).Scan(
		&will.ID,
		&will.UserID,
		&will.Status,
		&will.Version,
		&will.DormancyPeriodDays,
		&will.GracePeriodDays,
		&will.PolicyVersionAccepted,
		&will.ConsentAcceptedAt,
		&will.CreatedAt,
		&will.UpdatedAt,
	)
	if err != nil {
		return domain.DigitalWill{}, mapNotFound(err)
	}

	will.ReleaseCategories, err = loadCategories(ctx, s.pool, `
		SELECT category FROM waaris.will_release_preferences WHERE will_id = $1 ORDER BY category
	`, will.ID)
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

	var will domain.DigitalWill
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, status, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
		FROM waaris.digital_wills
		WHERE user_id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(
		&will.ID,
		&will.UserID,
		&will.Status,
		&will.Version,
		&will.DormancyPeriodDays,
		&will.GracePeriodDays,
		&will.PolicyVersionAccepted,
		&will.ConsentAcceptedAt,
		&will.CreatedAt,
		&will.UpdatedAt,
	)
	if err != nil {
		return domain.DigitalWill{}, mapNotFound(err)
	}

	nextVersion := will.Version + 1
	versionID := uuid.New()
	consentID := uuid.New()

	err = tx.QueryRow(ctx, `
		UPDATE waaris.digital_wills
		SET status = $2, current_version = $3, dormancy_days = $4, grace_days = $5, policy_version = $6, consent_accepted_at = $7, updated_at = $7
		WHERE id = $1
		RETURNING id, user_id, status, current_version, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at, updated_at
	`, will.ID, input.Status, nextVersion, input.DormancyPeriodDays, input.GracePeriodDays, input.PolicyVersionAccepted, now).Scan(
		&will.ID,
		&will.UserID,
		&will.Status,
		&will.Version,
		&will.DormancyPeriodDays,
		&will.GracePeriodDays,
		&will.PolicyVersionAccepted,
		&will.ConsentAcceptedAt,
		&will.CreatedAt,
		&will.UpdatedAt,
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
		INSERT INTO waaris.will_versions (
			id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
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

	will.ReleaseCategories = domain.CloneCategories(input.ReleaseCategories)
	if err = tx.Commit(ctx); err != nil {
		return domain.DigitalWill{}, err
	}
	return will, nil
}

func (s *Store) DeleteWill(ctx context.Context, userID uuid.UUID, now time.Time) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE waaris.digital_wills
		SET deleted_at = $2, updated_at = $2
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) WillHistory(ctx context.Context, userID uuid.UUID) ([]domain.WillVersion, error) {
	var willID uuid.UUID
	if err := s.pool.QueryRow(ctx, `
		SELECT id
		FROM waaris.digital_wills
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID).Scan(&willID); err != nil {
		return nil, mapNotFound(err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, will_id, user_id, version, status, dormancy_days, grace_days, policy_version, consent_accepted_at, created_at
		FROM waaris.will_versions
		WHERE will_id = $1
		ORDER BY version DESC
	`, willID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.WillVersion
	for rows.Next() {
		var item domain.WillVersion
		if err = rows.Scan(
			&item.ID,
			&item.WillID,
			&item.UserID,
			&item.Version,
			&item.Status,
			&item.DormancyPeriodDays,
			&item.GracePeriodDays,
			&item.PolicyVersionAccepted,
			&item.ConsentAcceptedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.ReleaseCategories, err = loadCategories(ctx, s.pool, `
			SELECT category
			FROM waaris.will_version_release_preferences
			WHERE will_version_id = $1
			ORDER BY category
		`, item.ID)
		if err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func insertReleasePreferences(ctx context.Context, q queryer, table, keyColumn string, key uuid.UUID, categories []domain.ReleaseCategory, now time.Time) error {
	for _, category := range categories {
		if _, err := q.Exec(ctx, `
			INSERT INTO `+table+` (`+keyColumn+`, category, created_at)
			VALUES ($1, $2, $3)
		`, key, category, now); err != nil {
			return err
		}
	}
	return nil
}

type categoryQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadCategories(ctx context.Context, q categoryQueryer, query string, key uuid.UUID) ([]domain.ReleaseCategory, error) {
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
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
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
