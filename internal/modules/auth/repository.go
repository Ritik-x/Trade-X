package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateUser(
	ctx context.Context,
	user *User,
) error {

	query := `
		INSERT INTO users (
			full_name,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			full_name,
			email,
			password_hash,
			status,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.FullName,
		user.Email,
		user.PasswordHash,
	).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailAlreadyExists
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*User, error) {

	query := `
		SELECT
			id,
			full_name,
			email,
			password_hash,
			status,
			created_at,
			updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	user := &User{}

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByID(
	ctx context.Context,
	userID uuid.UUID,
) (*User, error) {

	query := `
		SELECT
			id,
			full_name,
			email,
			password_hash,
			status,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

func (r *SessionRepository) Rotate(
	ctx context.Context,
	oldTokenHash string,
	newTokenHash string,
	newExpiresAt time.Time,
) (*Session, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin session rotation: %w", err)
	}
	defer tx.Rollback(ctx)

	session := &Session{}

	err = tx.QueryRow(ctx, `
        SELECT id, user_id
        FROM sessions
        WHERE refresh_token_hash = $1
          AND revoked_at IS NULL
          AND expires_at > NOW()
        FOR UPDATE
    `, oldTokenHash).Scan(&session.ID, &session.UserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("find refresh session: %w", err)
	}

	_, err = tx.Exec(ctx, `
        UPDATE sessions
        SET revoked_at = NOW(), updated_at = NOW()
        WHERE id = $1
    `, session.ID)
	if err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO sessions (user_id, refresh_token_hash, expires_at)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at
    `, session.UserID, newTokenHash, newExpiresAt).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create rotated session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit session rotation: %w", err)
	}

	return session, nil
}
