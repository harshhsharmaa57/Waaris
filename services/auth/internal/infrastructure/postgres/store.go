package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waaris/waaris/services/auth/internal/domain"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (domain.User, error) {
	user := domain.User{ID: uuid.New()}
	err := s.pool.QueryRow(ctx, `INSERT INTO waaris.users (id, email, password_hash, display_name) VALUES ($1, $2, $3, $4) RETURNING email, display_name, created_at, updated_at`, user.ID, email, passwordHash, displayName).Scan(&user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.User{}, domain.ErrEmailTaken
	}
	return user, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.UserWithPassword, error) {
	var user domain.UserWithPassword
	err := s.pool.QueryRow(ctx, `SELECT id, email, password_hash, display_name, created_at, updated_at FROM waaris.users WHERE email = $1`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	return user, mapNotFound(err)
}

func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `SELECT id, email, display_name, created_at, updated_at FROM waaris.users WHERE id = $1`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	return user, mapNotFound(err)
}

func (s *Store) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string) (domain.User, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `UPDATE waaris.users SET display_name = $2, updated_at = NOW() WHERE id = $1 RETURNING id, email, display_name, created_at, updated_at`, id, displayName).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	return user, mapNotFound(err)
}

func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	command, err := s.pool.Exec(ctx, `DELETE FROM waaris.users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CreateRefreshToken(ctx context.Context, token domain.RefreshToken) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO waaris.refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`, token.ID, token.UserID, token.Hash, token.ExpiresAt, token.CreatedAt)
	return err
}

func (s *Store) RotateRefreshToken(ctx context.Context, hash string, nextID uuid.UUID, nextHash string, expiresAt, now time.Time) (domain.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	var user domain.User
	err = tx.QueryRow(ctx, `SELECT u.id, u.email, u.display_name, u.created_at, u.updated_at FROM waaris.refresh_tokens r JOIN waaris.users u ON u.id = r.user_id WHERE r.token_hash = $1 AND r.revoked_at IS NULL AND r.expires_at > $2 FOR UPDATE`, hash, now).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return domain.User{}, mapRefreshNotFound(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE waaris.refresh_tokens SET revoked_at = $2 WHERE token_hash = $1`, hash, now); err != nil {
		return domain.User{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO waaris.refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`, nextID, user.ID, nextHash, expiresAt, now); err != nil {
		return domain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Store) RevokeRefreshToken(ctx context.Context, hash string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE waaris.refresh_tokens SET revoked_at = $2 WHERE token_hash = $1 AND revoked_at IS NULL`, hash, now)
	return err
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
func mapRefreshNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidRefreshToken
	}
	return err
}
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
