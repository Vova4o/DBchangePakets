package mongodb

import (
	"context"
	"errors"
	"fmt"

	"dbchangepakets/internal/domain"
	"dbchangepakets/internal/user"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository implements user.UserRepository for MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(client *mongo.Client, dbName, collName string) *UserRepository {
	return &UserRepository{
		collection: client.Database(dbName).Collection(collName),
	}
}

// Create inserts a user document.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	_, err := r.collection.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("create user: %w", user.ErrEmailTaken)
		}
		return fmt.Errorf("create user mongo: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("get user by id: %w", user.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by id mongo: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by Email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("get user by email: %w", user.ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by email mongo: %w", err)
	}
	return &u, nil
}

// EnsureIndexes creates unique index on email.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("create unique index: %w", err)
	}
	return nil
}
