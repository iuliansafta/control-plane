package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"` // Never expose token hash
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	IsRevoked bool       `json:"is_revoked"`
}

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at, is_revoked
	`, userID, tokenHash, expiresAt).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.IsRevoked,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, is_revoked
		FROM refresh_tokens
		WHERE token_hash = $1 AND is_revoked = false
	`, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
		&token.IsRevoked,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &token, nil
}

// GetByUserID retrieves all active refresh tokens for a user
func (r *RefreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, is_revoked
		FROM refresh_tokens
		WHERE user_id = $1 AND is_revoked = false
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []RefreshToken
	for rows.Next() {
		var token RefreshToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.ExpiresAt,
			&token.CreatedAt,
			&token.RevokedAt,
			&token.IsRevoked,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

// Revoke marks a refresh token as revoked
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens 
		SET is_revoked = true, revoked_at = $1 
		WHERE token_hash = $2
	`, now, tokenHash)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// RevokeByID marks a refresh token as revoked by ID
func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens 
		SET is_revoked = true, revoked_at = $1 
		WHERE id = $2
	`, now, id)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// RevokeAllByUserID revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens 
		SET is_revoked = true, revoked_at = $1 
		WHERE user_id = $2 AND is_revoked = false
	`, now, userID)

	return err
}

// DeleteExpired deletes all expired refresh tokens (cleanup)
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM refresh_tokens 
		WHERE expires_at < NOW()
	`)
	return err
}

// DeleteByUserID deletes all refresh tokens for a user
func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM refresh_tokens 
		WHERE user_id = $1
	`, userID)
	return err
}

// Count returns the total number of active refresh tokens
func (r *RefreshTokenRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_tokens 
		WHERE is_revoked = false AND expires_at > NOW()
	`).Scan(&count)
	return count, err
}

// CountByUserID returns the number of active refresh tokens for a user
func (r *RefreshTokenRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_tokens 
		WHERE user_id = $1 AND is_revoked = false AND expires_at > NOW()
	`, userID).Scan(&count)
	return count, err
}
