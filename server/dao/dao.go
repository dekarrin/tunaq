// Package dao provides data access objects for use in the TunaQuest server.
package dao

import (
	"context"

	"github.com/google/uuid"
)

type Command struct {
}

type Game struct {
}

type Session struct {
}

type UserRepository interface {

	// Create creates a new User. All attributes except for auto-generated
	// fields (ID) are taken from the provided User.
	Create(ctx context.Context, user User) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Delete(ctx context.Context, id uuid.UUID) (User, error)
}

type User struct {
	ID       uuid.UUID
	Username string
	Password string
}
