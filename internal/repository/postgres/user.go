package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"dbchangepakets/internal/domain"
	"dbchangepakets/internal/user"
)

// UserRepository implements user.UserRepository for PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Username, u.Email, u.CreatedAt)
	if err != nil {
		// Detect duplicate key constraint violation (PostgreSQL code 23505)
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique constraint") {
			return fmt.Errorf("create user: %w", user.ErrEmailTaken)
		}
		return fmt.Errorf("create user postgres: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, username, email, created_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get user by id: %w", user.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by id postgres: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by Email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, created_at
		FROM users
		WHERE email = $1
	`
	var u domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get user by email: %w", user.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by email postgres: %w", err)
	}
	return &u, nil
}


