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
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/dekarrin/tunaq/server/tunas"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
//  X POST   /users          - create a new user account (auth required)
//  X GET    /users          - get all users (auth required, with filter)
//  X GET    /users/{id}     - get info on a user (auth required)
//  X PUT    /users/{id}     - create an existing user
//  X PATCH  /users/{id}     - Update a user
//  X DELETE /users/{id}     - delete a user (auth required)
//  - GET    /info           - get version info on the game and engine itself.
//

// TunaQuestServer is an HTTP REST server that provides TunaQuest games and
// associated resources. The zero-value of a TunaQuestServer should not be used
// directly; call New() to get one ready for use.
type TunaQuestServer struct {
	router        chi.Router
	api           API
	srv           *http.ServeMux
	db            dao.Store
	unauthedDelay time.Duration
	jwtSecret     []byte
}

// New creates a new TunaQuestServer. If cfg is non-nil, any set values in it
// are used to configure the behavior of the server.
func New(cfg *Config) (TunaQuestServer, error) {
	// check config
	if cfg == nil {
		cfg = &Config{}
	}
	*cfg = cfg.FillDefaults()
	if err := cfg.Validate(); err != nil {
		return TunaQuestServer{}, fmt.Errorf("config: %w", err)
	}

	// connect DB
	db, err := cfg.DB.Connect()
	if err != nil {
		return TunaQuestServer{}, nil
	}

	tqAPI := API{
		Secret:      cfg.TokenSecret,
		UnauthDelay: cfg.UnauthDelay(),
		Backend: tunas.Service{
			DB: db,
		},
	}

	router := newRouter(tqAPI)

	return TunaQuestServer{
		api:    tqAPI,
		router: router,
	}, nil
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
	log.Printf("INFO  Listening on %s", listenAddress)
	log.Fatalf("FATAL %v", http.ListenAndServe(listenAddress, tqs.router))
}

// Login verifies the provided username and password against the existing user
// in persistence and returns that user if they match. Returns the user entity
// from the persistence layer that the username and password are valid for.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the credentials do not match
// a user or if the password is incorrect, it will match ErrBadCredentials. If
// the error occured due to an unexpected problem with the DB, it will match
// serr.ErrDB.
func (tqs TunaQuestServer) Login(ctx context.Context, username string, password string) (dao.User, error) {
	user, err := tqs.db.Users().GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrBadCredentials
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	// verify password
	bcryptHash, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		return dao.User{}, err
	}

	err = bcrypt.CompareHashAndPassword(bcryptHash, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return dao.User{}, serr.ErrBadCredentials
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	// successful login; update the DB
	user.LastLoginTime = time.Now()
	user, err = tqs.db.Users().Update(ctx, user.ID, user)
	if err != nil {
		return dao.User{}, serr.WrapDB("cannot update user login time", err)
	}

	return user, nil
}

// Logout marks the user with the given ID as having logged out, invalidating
// any login that may be active. Returns the user entity that was logged out.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the user doesn't exist, it
// will match serr.ErrNotFound. If the error occured due to an unexpected
// problem with the DB, it will match serr.ErrDB.
func (tqs TunaQuestServer) Logout(ctx context.Context, who uuid.UUID) (dao.User, error) {
	existing, err := tqs.db.Users().GetByID(ctx, who)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not retrieve user", err)
	}

	existing.LastLogoutTime = time.Now()

	updated, err := tqs.db.Users().Update(ctx, existing.ID, existing)
	if err != nil {
		return dao.User{}, serr.WrapDB("could not update user", err)
	}

	return updated, nil
}

// DeleteUser deletes the user with the given ID. It returns the deleted user
// just after they were deleted.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with that username
// exists, it will match serr.ErrNotFound. If the error occured due to an
// unexpected problem with the DB, it will match serr.ErrDB. Finally, if there
// is an issue with one of the arguments, it will match serr.ErrBadArgument.
func (tqs TunaQuestServer) DeleteUser(ctx context.Context, id string) (dao.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	user, err := tqs.db.Users().Delete(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not delete user", err)
	}

	return user, nil
}

// GetUser returns the user with the given ID.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with that ID exists,
// it will match serr.ErrNotFound. If the error occured due to an unexpected
// problem with the DB, it will match serr.ErrDB. Finally, if there is an issue
// with one of the arguments, it will match serr.ErrBadArgument.
func (tqs TunaQuestServer) GetUser(ctx context.Context, id string) (dao.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	user, err := tqs.db.Users().GetByID(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not get user", err)
	}

	return user, nil
}

// GetAllUsers returns all users currently in persistence.
func (tqs TunaQuestServer) GetAllUsers(ctx context.Context) ([]dao.User, error) {
	users, err := tqs.db.Users().GetAll(ctx)
	if err != nil {
		return nil, serr.WrapDB("", err)
	}

	return users, nil
}

// UpdatePassword sets the password of the user with the given ID to the new
// password. The new password cannot be empty. Returns the updated user.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with the given ID
// exists, it will match serr.ErrNotFound. If the error occured due to an
// unexpected problem with the DB, it will match serr.ErrDB. Finally, if one of
// the arguments is invalid, it will match serr.ErrBadArgument.
func (tqs TunaQuestServer) UpdatePassword(ctx context.Context, id, password string) (dao.User, error) {
	if password == "" {
		return dao.User{}, serr.New("password cannot be empty", serr.ErrBadArgument)
	}
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	existing, err := tqs.db.Users().GetByID(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("no user with that ID exists", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, serr.New("password is too long", err, serr.ErrBadArgument)
		} else {
			return dao.User{}, serr.New("password could not be encrypted", err)
		}
	}

	storedPass := base64.StdEncoding.EncodeToString(passHash)

	existing.Password = storedPass

	updated, err := tqs.db.Users().Update(ctx, uuidID, existing)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("no user with that ID exists", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("could not update user", err)
	}

	return updated, nil
}

// UpdateUser sets the properties of the user with the given ID to the
// properties in the given user. All the given properties of the user will
// overwrite the existing ones. Returns the updated user.
//
// This function cannot be used to update the password. Use UpdatePassword for
// that.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If a user with that username or
// ID (if they are changing) is already present, it will match
// serr.ErrAlreadyExists. If no user with the given ID exists, it will match
// serr.ErrNotFound. If the error occured due to an unexpected problem with the
// DB, it will match serr.ErrDB. Finally, if one of the arguments is invalid, it
// will match serr.ErrBadArgument.
func (tqs TunaQuestServer) UpdateUser(ctx context.Context, curID, newID, username, email string, role dao.Role) (dao.User, error) {
	var err error

	if username == "" {
		return dao.User{}, serr.New("username cannot be blank", err, serr.ErrBadArgument)
	}

	var storedEmail *mail.Address
	if email != "" {
		storedEmail, err = mail.ParseAddress(email)
		if err != nil {
			return dao.User{}, serr.New("email is not valid", err, serr.ErrBadArgument)
		}
	}

	uuidCurID, err := uuid.Parse(curID)
	if err != nil {
		return dao.User{}, serr.New("current ID is not valid", serr.ErrBadArgument)
	}
	uuidNewID, err := uuid.Parse(newID)
	if err != nil {
		return dao.User{}, serr.New("new ID is not valid", serr.ErrBadArgument)
	}

	daoUser, err := tqs.db.Users().GetByID(ctx, uuidCurID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("user not found", serr.ErrNotFound)
		}
	}

	if curID != newID {
		_, err := tqs.db.Users().GetByID(ctx, uuidNewID)
		if err == nil {
			return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
		} else if !errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.WrapDB("", err)
		}
	}
	if daoUser.Username != username {
		_, err := tqs.db.Users().GetByUsername(ctx, username)
		if err == nil {
			return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
		} else if !errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.WrapDB("", err)
		}
	}

	daoUser.Email = storedEmail
	daoUser.ID = uuidNewID
	daoUser.Username = username
	daoUser.Role = role

	updatedUser, err := tqs.db.Users().Update(ctx, uuidCurID, daoUser)
	if err != nil {
		if errors.Is(err, dao.ErrConstraintViolation) {
			return dao.User{}, serr.New("a user with that ID/username already exists", serr.ErrAlreadyExists)
		} else if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("user not found", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	return updatedUser, nil
}

// CreateUser creates a new user with the given username, password, and email
// combo. Returns the newly-created user as it exists after creation.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If a user with that username is
// already present, it will match serr.ErrAlreadyExists. If the error occured
// due to an unexpected problem with the DB, it will match serr.ErrDB. Finally,
// if one of the arguments is invalid, it will match serr.ErrBadArgument.
func (tqs TunaQuestServer) CreateUser(ctx context.Context, username, password, email string, role dao.Role) (dao.User, error) {
	var err error
	if username == "" {
		return dao.User{}, serr.New("username cannot be blank", err, serr.ErrBadArgument)
	}
	if password == "" {
		return dao.User{}, serr.New("password cannot be blank", err, serr.ErrBadArgument)
	}

	var storedEmail *mail.Address
	if email != "" {
		storedEmail, err = mail.ParseAddress(email)
		if err != nil {
			return dao.User{}, serr.New("email is not valid", err, serr.ErrBadArgument)
		}
	}

	_, err = tqs.db.Users().GetByUsername(ctx, username)
	if err == nil {
		return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
	} else if !errors.Is(err, dao.ErrNotFound) {
		return dao.User{}, serr.WrapDB("", err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, serr.New("password is too long", err, serr.ErrBadArgument)
		} else {
			return dao.User{}, serr.New("password could not be encrypted", err)
		}
	}

	storedPass := base64.StdEncoding.EncodeToString(passHash)

	newUser := dao.User{
		Username: username,
		Password: storedPass,
		Email:    storedEmail,
		Role:     role,
	}

	user, err := tqs.db.Users().Create(ctx, newUser)
	if err != nil {
		if errors.Is(err, dao.ErrConstraintViolation) {
			return dao.User{}, serr.ErrAlreadyExists
		}
		return dao.User{}, serr.WrapDB("could not create user", err)
	}

	return user, nil
}
