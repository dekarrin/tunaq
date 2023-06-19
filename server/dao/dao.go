// Package dao provides data access objects for use in the TunaQuest server.
package dao

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/internal/game"
	"github.com/google/uuid"
)

// Store holds all the repositories.
type Store struct {
	Users         UserRepository
	Registrations RegistrationRepository
	Commands      CommnadRepository
	Games         GameRepository
	Sessions      SessionRepository
}

func (s Store) Close() []error {
	errs := []error{}

	errs = append(errs, s.Users.Close())

	return errs
}

type CommnadRepository interface {
	Create(ctx context.Context, reg Command) (Command, error)
	GetByID(ctx context.Context, id uuid.UUID) (Command, error)
	GetAll(ctx context.Context) ([]Command, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]Command, error)
	GetAllByDateRange(ctx context.Context, notBefore, notAfter time.Time) ([]Command, error)
	Update(ctx context.Context, id uuid.UUID, reg Command) (Command, error)
	Delete(ctx context.Context, id uuid.UUID) (Command, error)
	Close() error
}

type Command struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Created time.Time
	Command string
}

type GameRepository interface {
	Create(ctx context.Context, game Game) (Game, error)
	GetByID(ctx context.Context, id uuid.UUID) (Game, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]Game, error)
	GetAll(ctx context.Context) ([]Game, error)
	Update(ctx context.Context, id uuid.UUID, sesh Game) (Game, error)
	Delete(ctx context.Context, id uuid.UUID) (Game, error)
	Close() error
}

type Game struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Created time.Time
	Storage string
}

type SessionRepository interface {
	Create(ctx context.Context, sesh Session) (Session, error)
	GetByID(ctx context.Context, id uuid.UUID) (Session, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]Session, error)
	GetAllByGame(ctx context.Context, gameID uuid.UUID) ([]Session, error)
	GetAll(ctx context.Context) ([]Session, error)
	Update(ctx context.Context, id uuid.UUID, sesh Session) (Session, error)
	Delete(ctx context.Context, id uuid.UUID) (Session, error)
	Close() error
}

type Session struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	GameID  uuid.UUID
	Created time.Time
	State   *game.State
}

type RegistrationRepository interface {
	Create(ctx context.Context, reg Registration) (Registration, error)
	GetByID(ctx context.Context, id uuid.UUID) (Registration, error)
	GetAll(ctx context.Context) ([]Registration, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]Registration, error)
	Update(ctx context.Context, id uuid.UUID, reg Registration) (Registration, error)
	Delete(ctx context.Context, id uuid.UUID) (Registration, error)
	Close() error
}

type Registration struct {
	ID      uuid.UUID // PK, NOT NULL
	UserID  uuid.UUID // FK (Many-to-One User.ID), NOT NULL
	Code    string    // NOT NULL
	Created time.Time // NOT NULL DEFAULT NOW()
	Expires time.Time // NOT NULL
}

type UserRepository interface {

	// Create creates a new User. All attributes except for auto-generated
	// fields are taken from the provided User.
	Create(ctx context.Context, user User) (User, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	GetAll(ctx context.Context) ([]User, error)
	Update(ctx context.Context, id uuid.UUID, user User) (User, error)
	Delete(ctx context.Context, id uuid.UUID) (User, error)

	// Close closes the connection.
	Close() error
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
	ID             uuid.UUID     // PK, NOT NULL
	Username       string        // UNIQUE, NOT NULL
	Password       string        // NOT NULL
	Email          *mail.Address // NOT NULL
	Role           Role          // NOT NULL
	LastLogoutTime time.Time     // NOT NULL DEFAULT NOW()
}
