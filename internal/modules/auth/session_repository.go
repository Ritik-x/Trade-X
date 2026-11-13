package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

// Create stores a new authenticated session.
func (r *SessionRepository) Create(
	ctx context.Context,
	session *Session,
) error {

	query := `
		INSERT INTO sessions (
			user_id,
			refresh_token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
	).Scan(
		&session.ID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetActiveByTokenHash retrieves a valid, non-revoked session.
func (r *SessionRepository) GetActiveByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*Session, error) {

	query := `
		SELECT
			id,
			user_id,
			refresh_token_hash,
			expires_at,
			revoked_at,
			created_at,
			updated_at
		FROM sessions
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	session := &Session{}

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("get active session: %w", err)
	}

	return session, nil
}

// Revoke marks a session as revoked.
func (r *SessionRepository) Revoke(
	ctx context.Context,
	sessionID uuid.UUID,
) error {

	query := `
		UPDATE sessions
		SET revoked_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
