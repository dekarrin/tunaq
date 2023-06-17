package server

import (
	"context"
	"encoding/base64"
	"errors"
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

var (
	ErrBadCredentials = errors.New("the supplied username/password combo is incorrect")
	ErrPermissions    = errors.New("you don't have permission to do that")
	ErrNotFound       = errors.New("the requested entity could not be found")
	ErrAlreadyExists  = errors.New("resource with same identifying information already exists")
	ErrDB             = errors.New("an error occured with the DB")
	ErrBadArgument    = errors.New("one or more of the arguments is invalid")
)

// js site interface -> {"input": "TAKE SPOON", "session": "alkdf803=="} -> server
// server -> {"output": "You take the spoon"} -> js site interface
//
// server:
//  X POST   /login          - accepts user and password and returns a jwt.
//  X DELETE /login/{id}     - ends user authentication session and destroyes the jwt.
//  X POST   /tokens         - refreshes the token without requiring credentials (requires auth)
//	- POST   /commands       - accepts command input from user, requires session token
//  - GET    /commands       - return command history, requires session token
//  - GET    /commands/{id}  - gets a particular command from history
//	- POST   /sessions       - create a new game session. (auth not required) Requires the name of the world file, or takes the default on disk
//	- GET    /sessions/{id}  - get info on a game (if it's yours) (auth not required, session stored in cookie)
//  - POST   /games          - create a new game (auth required), file upload required
//  - GET    /games/{id}     - get info on a game (auth not required)
//	- GET    /games          - get info on all games (auth not required)
//  - DELETE /games/{id}     - delete a game (auth required)
//  - POST   /registrations  - request a new user account (auth not required)
//  - POST   /users          - create a new user account (auth required)
//  - GET    /users          - get all users (auth required)
//  - GET    /users/{id}     - get info on a user (auth required)
//  - PUT    /users/{id}     - Update a user
//  - PATCH  /users/{id}     - Update a user
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
// from the persistence layer that the username and password are valid for.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the credentials do not match
// a user or if the password is incorrect, it will match ErrBadCredentials. If
// the error occured due to an unexpected problem with the DB, it will match
// ErrDB.
func (tqs TunaQuestServer) Login(ctx context.Context, username string, password string) (dao.User, error) {
	user, err := tqs.db.Users.GetByUsername(ctx, username)
	if err != nil {
		if err == inmem.ErrNotFound {
			return dao.User{}, ErrBadCredentials
		}
		return dao.User{}, wrapDBError(err)
	}

	// verify password
	bcryptHash, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		return dao.User{}, err
	}

	err = bcrypt.CompareHashAndPassword(bcryptHash, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return dao.User{}, ErrBadCredentials
		}
		return dao.User{}, wrapDBError(err)
	}

	return user, nil
}

// Logout marks the user with the given ID as having logged out, invalidating
// any login that may be active. Returns the user entity that was logged out.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the user doesn't exist, it
// will match ErrNotFound.  If the error occured due to an unexpected problem
// with the DB, it will match ErrDB.
func (tqs TunaQuestServer) Logout(ctx context.Context, who uuid.UUID) (dao.User, error) {
	existing, err := tqs.db.Users.GetByID(ctx, who)
	if err != nil {
		if err == inmem.ErrNotFound {
			return dao.User{}, ErrNotFound
		}
		return dao.User{}, newError("could not retrieve user", err, ErrDB)
	}

	existing.LastLogoutTime = time.Now()

	updated, err := tqs.db.Users.Update(ctx, existing)
	if err != nil {
		return dao.User{}, newError("could not update user", err, ErrDB)
	}

	return updated, nil
}

// DeleteUser deletes the user with the given ID. It returns the deleted user
// just after they were deleted.

// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with that username
// exists, it will match ErrNotFound. If the error occured due to an unexpected
// problem with the DB, it will match ErrDB. Finally, if there is an issue with
// one of the arguments, it will match ErrBadArgument.
func (tqs TunaQuestServer) DeleteUser(ctx context.Context, id string) (dao.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, newError("ID is not valid", ErrBadArgument)
	}

	user, err := tqs.db.Users.Delete(ctx, uuidID)
	if err != nil {
		if err == inmem.ErrNotFound {
			return dao.User{}, ErrNotFound
		}
		return dao.User{}, newError("could not delete user", err, ErrDB)
	}

	return user, nil
}

// CreateUser creates a new user with the given username, password, and email
// combo. Returns the newly-created user as it exists after creation.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If a user with that username is
// already present, it will match ErrAlreadyExists. If the error occured due to
// an unexpected problem with the DB, it will match ErrDB. Finally, if one of
// the arguments is invalid, it will match ErrBadArgument.
func (tqs TunaQuestServer) CreateUser(ctx context.Context, username, password, email string, role dao.Role) (dao.User, error) {
	_, err := tqs.db.Users.GetByUsername(ctx, username)
	if err != inmem.ErrNotFound {
		return dao.User{}, newError("a user with that username already exists", ErrAlreadyExists)
	}

	var storedEmail *mail.Address
	if email != "" {
		storedEmail, err = mail.ParseAddress(email)
		if err != nil {
			return dao.User{}, newError("email is not valid", err, ErrBadArgument)
		}
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, newError("password is too long", err, ErrBadArgument)
		} else {
			return dao.User{}, newError("password could not be encrypted", err)
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
			return dao.User{}, ErrAlreadyExists
		}
		return dao.User{}, newError("could not create user", err, ErrDB)
	}

	return user, nil
}

// Error is an error in the server.
type Error struct {
	msg   string
	cause []error
}

func (e Error) Error() string {
	if e.msg == "" && e.cause != nil {
		return e.cause[0].Error()
	}

	if e.cause != nil {
		return e.msg + ": " + e.cause[0].Error()
	}

	return e.msg
}

func (e Error) Unwrap() []error {
	if len(e.cause) > 0 {
		return e.cause
	}
	return nil
}

func wrapDBError(err error) error {
	return Error{
		cause: []error{err, ErrDB},
	}
}

func wrapError(err error, others ...error) error {
	errObj := Error{}
	if len(others) > 0 {
		errObj.cause = make([]error, len(others)+1)
		copy(errObj.cause[1:], others)
	} else {
		errObj.cause = []error{err}
	}
	return errObj
}

func newError(msg string, causes ...error) error {
	err := Error{msg: msg}
	if len(causes) > 0 {
		err.cause = make([]error, len(causes))
		copy(err.cause, causes)
	}
	return err
}
