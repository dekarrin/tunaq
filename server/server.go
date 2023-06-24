package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/dao/inmem"
	"github.com/dekarrin/tunaq/server/dao/sqlite"
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

// DBType is the type of a Database connection.
type DBType string

func (dbt DBType) String() string {
	return string(dbt)
}

const (
	DatabaseNone     DBType = "none"
	DatabaseSQLite   DBType = "sqlite"
	DatabaseInMemory DBType = "inmem"
)

// ParseDBType parses a string found in a connection string into a DBType.
func ParseDBType(s string) (DBType, error) {
	sLower := strings.ToLower(s)

	switch sLower {
	case DatabaseSQLite.String():
		return DatabaseSQLite, nil
	case DatabaseInMemory.String():
		return DatabaseInMemory, nil
	default:
		return DatabaseNone, fmt.Errorf("DB type not one of 'sqlite' or 'inmem': %q", s)
	}
}

// Database contains configuration settings for connecting to a persistence
// layer.
type Database struct {
	// Type is the type of database the config refers to. It also determines
	// which of its other fields are valid.
	Type DBType

	// DataDir is the path on disk to a directory to use to store data in. This
	// is only applicable for certain DB types: SQLite.
	DataDir string
}

// Connect performs all logic needed to connect to the configured DB and
// initialize the store for use.
func (db Database) Connect() (dao.Store, error) {
	switch db.Type {
	case DatabaseInMemory:
		return inmem.NewDatastore(), nil
	case DatabaseSQLite:
		err := os.MkdirAll(db.DataDir, 0770)
		if err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}

		store, err := sqlite.NewDatastore(db.DataDir)
		if err != nil {
			return nil, fmt.Errorf("initialize sqlite: %w", err)
		}

		return store, nil
	case DatabaseNone:
		return nil, fmt.Errorf("cannot connect to 'none' DB")
	default:
		return nil, fmt.Errorf("unknown database type: %q", db.Type.String())
	}
}

// Validate returns an error if the Database does not have the correct fields
// set. Its type will be checked to ensure that it is a valid type to use and
// any fields necessary for connecting to that type of DB are also checked.
func (db Database) Validate() error {
	switch db.Type {
	case DatabaseInMemory:
		// nothing else to check
		return nil
	case DatabaseSQLite:
		if db.DataDir == "" {
			return fmt.Errorf("DataDir not set to path")
		}
		return nil
	case DatabaseNone:
		return fmt.Errorf("'none' DB is not valid")
	default:
		return fmt.Errorf("unknown database type: %q", db.Type.String())
	}
}

// ParseDBConnString parses a database connection string of the form
// "engine:params" (or just "engine" if no other params are required) into a
// valid Database config object. For example, "sqlite:/data" would give the DB
// type of DatabaseSQLite that stores persistence in files located in the given
// dir, and "inmem" would give the DB type of DatabaseInMemory.
func ParseDBConnString(s string) (Database, error) {
	var paramStr string
	dbParts := strings.SplitN(s, ":", 2)

	if len(dbParts) == 2 {
		paramStr = strings.TrimSpace(dbParts[1])
	}

	// parse the first section into a type, from there we can determine if
	// further params are required.
	dbEng, err := ParseDBType(strings.TrimSpace(dbParts[0]))
	if err != nil {
		return Database{}, fmt.Errorf("unsupported DB engine: %w", err)
	}

	switch dbEng {
	case DatabaseInMemory:
		// there cannot be any other options
		if paramStr != "" {
			return Database{}, fmt.Errorf("unsupported param(s) for in-memory DB engine: %s", paramStr)
		}

		return Database{Type: DatabaseInMemory}, nil
	case DatabaseSQLite:
		// there must be options
		if paramStr == "" {
			return Database{}, fmt.Errorf("sqlite DB engine requires path to data directory after ':'")
		}

		// the only option is the DB path, as long as the param str isn't
		// literally blank, it can be used.
		return Database{Type: DatabaseSQLite, DataDir: paramStr}, nil
	case DatabaseNone:
		// not allowed
		return Database{}, fmt.Errorf("cannot specify DB engine 'none' (perhaps you wanted 'inmem'?)")
	default:
		// unknown
		return Database{}, fmt.Errorf("unknown DB engine: %q", dbEng.String())
	}
}

// Config is a configuration for a server. It contains all parameters that can
// be used to configure the operation of a TunaQuestServer.
type Config struct {

	// TokenSecret is the secret used for signing tokens. If not provided, one
	// will be automatically generated. If a generated secret is used for token
	// signing, they will all become invalid at the completion of the server.
	TokenSecret []byte

	// Database is the configuration to use for connecting to the database. If
	// not provided, it will be set to a configuration for using an in-memory
	// persistence layer.
	DB Database

	// UnauthDelayMS is the amount of additional time to wait (in milliseconds)
	// before sending a response that indicates either that the client was
	// unauthorized or the client was unauthenticated. This is something of an
	// "anti-flood" for naive clients attempting non-parallel connections. If
	// not set it will default to 1 second. Set this to any negative number to
	// disable the delay.
	UnauthDelayMS int
}

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

// New creates a new TunaQuestServer. tokenSecret is used for signing issued
// tokens and dbPath is the path on disk to a data directory to use for
// persistence files; if empty the server will use an in-memory datastore
// instead.
func New(tokenSecret []byte, dbPath string) (TunaQuestServer, error) {
	// connect DB
	var err error
	var db dao.Store

	if dbPath != "" {
		db, err = sqlite.NewDatastore(dbPath)
		if err != nil {
			return TunaQuestServer{}, err
		}
	} else {
		db = inmem.NewDatastore()
	}

	tqs := TunaQuestServer{
		api: API{
			Backend: tunas.Service{},
		},
		jwtSecret:     tokenSecret,
		unauthedDelay: time.Second,
	}

	tqs.router = newRouter(&tqs)

	return tqs, nil
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
