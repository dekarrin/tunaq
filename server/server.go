package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/dao/inmem"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	fakeTestKey = []byte("TODO: DO NOT USE IN PROD, CHANGE ME")
)

type Error string

var (
	ErrBadCredentials Error = "The supplied username/password combo is incorrect"
)

func (e Error) Error() string {
	return string(e)
}

// NetworkJSONCommandReader reads commands
type NetworkJSONCommandReader struct {
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

type TunaQuestServer struct {
	srv *http.ServeMux

	users dao.UserRepository
}

type LoginResponse struct {
	Token string `json:"token"`
}

func New() TunaQuestServer {
	tqs := TunaQuestServer{
		srv:   http.NewServeMux(),
		users: inmem.NewUsersRepository(),
	}

	tqs.srv.HandleFunc("/login/", tqs.handlePathLogin)

	return tqs
}

func (tqs TunaQuestServer) ServeForever() {

}

func (tqs TunaQuestServer) handlePathLogin(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/login/" || req.URL.Path == "/login" {
		if req.Method == http.MethodPost {
			tok, err := tqs.Login()
			if err != nil {
				if err == ErrBadCredentials {
					log.Printf("ERROR: HTTP-401: %s", ErrBadCredentials)
					w.Header().Set("WWW-Authenticate", "Basic realm=\"TunaQuest server\", charset=\"utf-8\"")
					http.Error(w, "Incorrect username or password", http.StatusUnauthorized)
					return
				} else {
					log.Printf("ERROR: HTTP-500: %s", err.Error())
					http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
					return
				}
			}

			resp := LoginResponse{Token: tok.Raw}
			w.WriteHeader(http.StatusCreated)
			renderJSON(w, resp)

		} else {
			http.Error(w, fmt.Sprintf("HTTP %v method is not valid for %s", req.Method, req.URL.Path), http.StatusMethodNotAllowed)
			return
		}
	} else {
		// check for /login/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			http.Error(w, "The requested resource was not found", http.StatusNotFound)
			return
		}

		if req.Method == http.MethodDelete {
			id, err := uuid.Parse(pathParts[1])
			if err != nil {
				http.Error(w, "The requested resource was not found", http.StatusNotFound)
				return
			}
			err = tqs.Logout(nil, id)
			if err != nil {
				http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, fmt.Sprintf("HTTP %v method is not valid for %s", req.Method, req.URL.Path), http.StatusMethodNotAllowed)
			return
		}
	}
}

func (tqs TunaQuestServer) Login(username string, password string) (*jwt.Token, error) {

}

func (tqs TunaQuestServer) Logout(auth *jwt.Token, who uuid.UUID) error {

}

func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
