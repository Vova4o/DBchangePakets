package domain

import (
	"time"
)

// User represents the core user domain model.
type User struct {
	ID        string    `json:"id" bson:"_id"`
	Username  string    `json:"username" bson:"username"`
	Email     string    `json:"email" bson:"email"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
