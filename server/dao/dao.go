// Package dao provides data access objects for use in the TunaQuest server.
package dao

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store holds all the repositories.
type Store struct {
	Users UserRepository
}

type Command struct {
}

type Game struct {
}

type Session struct {
}

type UserRepository interface {

	// Create creates a new User. All attributes except for auto-generated
	// fields are taken from the provided User.
	Create(ctx context.Context, user User) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Update(ctx context.Context, id uuid.UUID, user User) (User, error)
	Delete(ctx context.Context, id uuid.UUID) (User, error)
}

type Role int

const (
	Guest Role = iota
	Unverified
	Normal

	Admin Role = 100
)

func (r Role) String() string {
	switch r {
	case Guest:
		return "guest"
	case Unverified:
		return "unverified"
	case Normal:
		return "normal"
	case Admin:
		return "admin"
	default:
		return fmt.Sprintf("Role(%d)", r)
	}
}

func ParseRole(s string) (Role, error) {
	check := strings.ToLower(s)
	switch check {
	case "guest":
		return Guest, nil
	case "unverified":
		return Unverified, nil
	case "normal":
		return Normal, nil
	case "admin":
		return Admin, nil
	default:
		return Guest, fmt.Errorf("must be one of 'guest', 'unverified', 'normal', or 'admin'")
	}
}

type User struct {
	ID             uuid.UUID
	Username       string
	Password       string
	Email          *mail.Address
	Role           Role
	LastLogoutTime time.Time
}
