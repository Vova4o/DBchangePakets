package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"dbchangepakets/internal/domain"
)

// Common domain and service errors.
var (
	ErrEmailTaken   = errors.New("email is already registered")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidInput = errors.New("invalid input data")
)

// UserRepository defines the database contracts needed by the UserService.
// This interface is defined on the consumer side (here) to ensure decoupling.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// Service implements the user business logic.
type Service struct {
	repo UserRepository
}

// NewService creates a new Service.
func NewService(repo UserRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// RegisterRequest defines the input payload for registering a user.
type RegisterRequest struct {
	Username string
	Email    string
}

// Register validates request data, checks for duplicate emails, and creates a user.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if username == "" {
		return nil, fmt.Errorf("%w: username cannot be empty", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	// Check if email already exists
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("register user lookup: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: email %q is already taken", ErrEmailTaken, email)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("register user generate id: %w", err)
	}

	u := &domain.User{
		ID:        id,
		Username:  username,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("register user create: %w", err)
	}

	return u, nil
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id cannot be empty", ErrInvalidInput)
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user lookup: %w", err)
	}

	return u, nil
}

// generateID creates a hex-encoded unique string for user IDs.
func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("random read: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
