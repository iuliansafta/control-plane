package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// User represents a user in the database
type User struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Never expose password hash
	IsActive      bool       `json:"is_active"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// UserRepository handles user database operations
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, name, email, passwordHash string) (*User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash, is_active, email_verified, created_at, last_login, updated_at
	`, name, email, passwordHash).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.LastLogin,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, is_active, email_verified, created_at, last_login, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.LastLogin,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, is_active, email_verified, created_at, last_login, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.LastLogin,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE users 
		SET last_login = $1 
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

// UpdateActive updates the user's active status
func (r *UserRepository) UpdateActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users 
		SET is_active = $1 
		WHERE id = $2
	`, isActive, id)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// UpdateEmailVerified updates the user's email verification status
func (r *UserRepository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users 
		SET email_verified = $1 
		WHERE id = $2
	`, verified, id)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// UpdatePassword updates the user's password hash
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users 
		SET password_hash = $1 
		WHERE id = $2
	`, passwordHash, id)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// List retrieves all users (paginated)
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, email, password_hash, is_active, email_verified, created_at, last_login, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.PasswordHash,
			&user.IsActive,
			&user.EmailVerified,
			&user.CreatedAt,
			&user.LastLogin,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
