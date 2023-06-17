package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/dao/inmem"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Error string

var (
	ErrBadCredentials Error = "The supplied username/password combo is incorrect"
	ErrPermissions    Error = "You don't have permission to do that"
	ErrInvalidLogin   Error = "You don't appear to be logged in"
	ErrNotFound       Error = "The requested entity could not be found"
)

func (e Error) Error() string {
	return string(e)
}

// js site interface -> {"input": "TAKE SPOON", "session": "alkdf803=="} -> server
// server -> {"output": "You take the spoon"} -> js site interface
//
// server:
//  - POST   /login          - accepts user and password and returns a jwt.
//  - DELETE /login/{id}     - ends user authentication session and destroyes the jwt.
//	- POST   /commands       - accepts command input from user, requires session token
//  - GET    /commands       - return command history, requires session token
//  - GET    /commands/{id}  - gets a particular command from history
//	- POST   /sessions       - create a new game session. (auth not required) Requires the name of the world file, or takes the default on disk
//	- GET    /sessions/{id}  - get info on a game (if it's yours) (auth not required, session stored in cookie)
//  - POST   /games          - create a new game (auth required), file upload required
//  - GET    /games/{id}     - get info on a game (auth not required)
//	- GET    /games          - get info on all games (auth not required)
//  - DELETE /games/{id}     - delete a game (auth required)
//  - POST   /users          - create a new account (auth not required)
//  - GET    /users/{id}     - get info on a user (auth required)
//  - GET    /users          - get all users (auth required)
//  - DELETE /users/{id}     - delete a user (auth required)
//  - GET    /info           - get version info on the game and engine itself.
//

// TunaQuestServer is an HTTP REST server that provides TunaQuest games and
// associated resources. The zero-value of a TunaQuestServer should not be used
// directly; call New() to get one ready for use.
type TunaQuestServer struct {
	srv       *http.ServeMux
	db        dao.Store
	jwtSecret []byte
}

// New creates a new TunaQuestServer that uses the given JWT secret for securing
// logins.
func New(tokenSecret []byte) TunaQuestServer {
	tqs := TunaQuestServer{
		srv: http.NewServeMux(),
		db: dao.Store{
			Users: inmem.NewUsersRepository(),
		},
		jwtSecret: tokenSecret,
	}

	tqs.initHandlers()

	return tqs
}

// ServeForever begins listening on the given address and port for HTTP REST
// client requests. If address is kept as "", it will default to "localhost". If
// port is less than 1, it will default to 8080.
func (tqs TunaQuestServer) ServeForever(address string, port int) {
	if address == "" {
		address = "localhost"
	}
	if port < 1 {
		port = 8080
	}

	listenAddress := fmt.Sprintf("%s:%d", address, port)
	log.Printf("INFO : Listening on %s", listenAddress)
	log.Fatalf("FATAL: %v", http.ListenAndServe(listenAddress, tqs.srv))
}

// Login verifies the provided username and password against the existing user
// in persistence and returns that user if they match. Returns the user entity
// from the persistence layer that the username and password are valid for. The
// returned error will be ErrBadCredentials if either no user with the given
// username exists or if the provided password is not correct.
func (tqs TunaQuestServer) Login(ctx context.Context, username string, password string) (dao.User, error) {
	user, err := tqs.db.Users.GetByUsername(ctx, username)
	if err != nil {
		if err == inmem.ErrNotFound {
			return dao.User{}, ErrBadCredentials
		}
	}

	// verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return dao.User{}, ErrBadCredentials
	}

	return user, nil
}

// Logout marks the user with the given ID as having logged out, invalidating
// any login that may be active. Returns the user entity that was logged out.
// The returned error will be ErrNotFound if the user doesn't exist, or some
// other non-nil error if there is a problem retrieving or updating the user.
func (tqs TunaQuestServer) Logout(ctx context.Context, who uuid.UUID) (dao.User, error) {
	existing, err := tqs.db.Users.GetByID(ctx, who)
	if err != nil {
		if err == inmem.ErrNotFound {
			return dao.User{}, ErrNotFound
		}
		return dao.User{}, fmt.Errorf("could not retrieve user: %w", err)
	}

	existing.LastLogoutTime = time.Now()

	updated, err := tqs.db.Users.Update(ctx, existing)
	if err != nil {
		return dao.User{}, fmt.Errorf("could not update user: %w", err)
	}

	return updated, nil
}

// CreateUser creates a new user with the given username, password, and email
// combo. Returns the newly-created user as it exists after creation.
func (tqs TunaQuestServer) CreateUser(ctx context.Context, username, password string, email string) (dao.User, error) {
	_, err := tqs.db.Users.GetByUsername(ctx, username)
	if err != inmem.ErrNotFound {
		return dao.User{}, fmt.Errorf("user already exists")
	}

	storedEmail, err := mail.ParseAddress(email)
	if err != nil {
		return dao.User{}, fmt.Errorf("email is not valid: %w", err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 20)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, fmt.Errorf("password is too long")
		} else {
			return dao.User{}, fmt.Errorf("password could not be encrypted: %w", err)
		}
	}

	storedPass := base64.StdEncoding.EncodeToString(passHash)

	newUser := dao.User{
		Username: username,
		Password: storedPass,
		Email:    storedEmail,
	}

	user, err := tqs.db.Users.Create(ctx, newUser)
	if err != nil {
		if err == inmem.ErrConstraintViolation {
			return dao.User{}, fmt.Errorf("user already exists")
		}
		return dao.User{}, fmt.Errorf("could not create user: %w", err)
	}

	return user, nil
}
