package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// APIKey API key in the database
type APIKey struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type APIKeyRepository struct {
	db *DB
}

func NewAPIKeyRepository(db *DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(name, keyHash string) (*APIKey, error) {
	var key APIKey
	err := r.db.QueryRow(context.Background(), `
		INSERT INTO api_keys (name, key_hash)
		VALUES ($1, $2)
		RETURNING id, user_id, name, key_hash, is_active, created_at, updated_at
	`, name, keyHash).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &key, nil
}

// CreateForUser creates a new API key associated with a user
func (r *APIKeyRepository) CreateForUser(ctx context.Context, userID uuid.UUID, name, keyHash string) (*APIKey, error) {
	var key APIKey
	err := r.db.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_hash)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, key_hash, is_active, created_at, updated_at
	`, userID, name, keyHash).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &key, nil
}

// GetByHash retrieves an API key by its hash
func (r *APIKeyRepository) GetByHash(keyHash string) (*APIKey, error) {
	var key APIKey
	err := r.db.QueryRow(context.Background(), `
		SELECT id, user_id, name, key_hash, is_active, created_at, updated_at
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &key, nil
}

// GetByID retrieves an API key by its ID
func (r *APIKeyRepository) GetByID(id uuid.UUID) (*APIKey, error) {
	var key APIKey
	err := r.db.QueryRow(context.Background(), `
		SELECT id, user_id, name, key_hash, is_active, created_at, updated_at
		FROM api_keys
		WHERE id = $1
	`, id).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &key, nil
}

// List retrieves all API keys
func (r *APIKeyRepository) List() ([]APIKey, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT id, user_id, name, key_hash, is_active, created_at, updated_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// ListByUserID retrieves all API keys for a specific user
func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, key_hash, is_active, created_at, updated_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.IsActive, &key.CreatedAt, &key.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// Delete deletes an API key by its ID
func (r *APIKeyRepository) Delete(id uuid.UUID) error {
	result, err := r.db.Exec(context.Background(), `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Deactivate deactivates an API key
func (r *APIKeyRepository) Deactivate(id uuid.UUID) error {
	result, err := r.db.Exec(context.Background(), `UPDATE api_keys SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Count returns the total number of API keys
func (r *APIKeyRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM api_keys`).Scan(&count)
	return count, err
}
