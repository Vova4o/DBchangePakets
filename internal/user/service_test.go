package user

import (
	"context"
	"errors"
	"sync"
	"testing"

	"dbchangepakets/internal/domain"
)

// fakeUserRepository implements UserRepository in-memory for testing.
type fakeUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (f *fakeUserRepository) Create(ctx context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.ID] = u
	return nil
}

func (f *fakeUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	u, ok := f.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func TestRegister(t *testing.T) {
	repo := newFakeUserRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// Test successful registration
	req := RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
	}
	u, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username alice, got %q", u.Username)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %q", u.Email)
	}
	if u.ID == "" {
		t.Error("expected generated user ID, got empty string")
	}

	// Test duplicate email
	_, err = svc.Register(ctx, req)
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected error %v, got %v", ErrEmailTaken, err)
	}

	// Test invalid email format
	invalidReq := RegisterRequest{
		Username: "bob",
		Email:    "invalid-email",
	}
	_, err = svc.Register(ctx, invalidReq)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected error %v, got %v", ErrInvalidInput, err)
	}

	// Test empty username
	emptyUserReq := RegisterRequest{
		Username: "",
		Email:    "bob@example.com",
	}
	_, err = svc.Register(ctx, emptyUserReq)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected error %v, got %v", ErrInvalidInput, err)
	}
}

func TestGetByID(t *testing.T) {
	repo := newFakeUserRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// Seed data
	u := &domain.User{
		ID:       "user-1",
		Username: "bob",
		Email:    "bob@example.com",
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	// Get user successfully
	found, err := svc.GetByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.Username != "bob" {
		t.Errorf("expected username bob, got %q", found.Username)
	}

	// Get non-existent user
	_, err = svc.GetByID(ctx, "user-999")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected error %v, got %v", ErrUserNotFound, err)
	}

	// Get with empty ID
	_, err = svc.GetByID(ctx, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected error %v, got %v", ErrInvalidInput, err)
	}
}
