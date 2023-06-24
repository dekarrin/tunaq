package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dekarrin/tunaq/server/tunas"
	"github.com/go-chi/chi/v5"
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

// TunaQuestRESTServer is an HTTP REST server that provides TunaQuest games and
// associated resources. The zero-value of a TunaQuestRESTServer should not be used
// directly; call New() to get one ready for use.
type TunaQuestRESTServer struct {
	router chi.Router
	api    API
}

// New creates a new TunaQuestServer. If cfg is non-nil, any set values in it
// are used to configure the behavior of the server.
func New(cfg *Config) (TunaQuestRESTServer, error) {
	// check config
	if cfg == nil {
		cfg = &Config{}
	}
	*cfg = cfg.FillDefaults()
	if err := cfg.Validate(); err != nil {
		return TunaQuestRESTServer{}, fmt.Errorf("config: %w", err)
	}

	// connect DB
	db, err := cfg.DB.Connect()
	if err != nil {
		return TunaQuestRESTServer{}, nil
	}

	tqAPI := API{
		Secret:      cfg.TokenSecret,
		UnauthDelay: cfg.UnauthDelay(),
		Backend: tunas.Service{
			DB: db,
		},
	}

	router := newRouter(tqAPI)

	return TunaQuestRESTServer{
		api:    tqAPI,
		router: router,
	}, nil
}

// Backend returns the service that is acting as the TunaQuestRESTServer's
// backend. This can be used to perform operations against the Backend directly
// instead of having to go through a REST endpoint.
func (tqs TunaQuestRESTServer) Backend() tunas.Service {
	return tqs.api.Backend
}

// ServeForever begins listening on the given address and port for HTTP REST
// client requests. If address is kept as "", it will default to "localhost". If
// port is less than 1, it will default to 8080.
func (tqs TunaQuestRESTServer) ServeForever(address string, port int) {
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
