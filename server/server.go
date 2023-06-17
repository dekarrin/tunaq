package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/dao/inmem"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	fakeTestKey = []byte("TODO: DO NOT USE IN PROD, CHANGE ME")
)

type Error string

var (
	ErrBadCredentials Error = "The supplied username/password combo is incorrect"
	ErrPermissions    Error = "You don't have permission to do that"
	ErrInvalidLogin   Error = "You don't appear to be logged in"
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

	db dao.Store
}

type LoginResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func New() TunaQuestServer {
	tqs := TunaQuestServer{
		srv: http.NewServeMux(),
		db: dao.Store{
			Users: inmem.NewUsersRepository(),
		},
	}

	tqs.srv.HandleFunc("/login/", tqs.handlePathLogin)

	return tqs
}

func (tqs TunaQuestServer) ServeForever() {

}

func (tqs TunaQuestServer) handlePathLogin(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/login/" || req.URL.Path == "/login" {
		if req.Method == http.MethodPost {
			loginData := LoginRequest{}
			err := parseJSON(req, &loginData)
			if err != nil {
				terminateWithError(w, req, http.StatusBadRequest, err.Error(), err.Error())
				return
			}

			user, err := tqs.Login(req.Context(), loginData.User, loginData.Password)
			if err != nil {
				if err == ErrBadCredentials {
					w.Header().Set("WWW-Authenticate", "Basic realm=\"TunaQuest server\", charset=\"utf-8\"")
					terminateWithError(w, req, http.StatusUnauthorized, err.Error(), err.Error())
					return
				} else {
					terminateWithError(w, req, http.StatusInternalServerError, "An internal server error occurred", err.Error())
					return
				}
			}

			// build the token
			// password is valid, generate token for user and return it.
			tok, err := generateJWTForUser(user)
			if err != nil {
				terminateWithError(w, req, http.StatusInternalServerError, "An internal server error occurred", "could not generate JWT: "+err.Error())
				return
			}

			resp := LoginResponse{Token: tok}
			terminateWithJSON(w, req, http.StatusCreated, resp, "user '"+user.Username+"' successfully logged in")
			return
		} else {
			terminateWithError(w, req, http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
			return
		}
	} else {
		// check for /login/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			http.Error(w, "The requested resource was not found", http.StatusNotFound)
			return
		}

		id, err := uuid.Parse(pathParts[1])
		if err != nil {
			http.Error(w, "The requested resource was not found", http.StatusNotFound)
			return
		}

		if req.Method == http.MethodDelete {
			// need to: get JWT
			// get WHO from request

			user, err := tqs.requireJWT(req.Context(), req)
			if err != nil {
				terminateWithError(w, req, http.StatusUnauthorized, "Valid bearer JWT token required", fmt.Sprintf("could not verify JWT: %s", err.Error()))
				return
			}

			// is the user trying to delete someone else? they'd betta be the admin if so!
			if id != user.ID && user.Role != dao.Admin {
				var otherUserStr string
				otherUser, err := tqs.db.Users.GetByID(req.Context(), id)
				// if there was another user, find out now
				if err != nil {
					otherUserStr = fmt.Sprintf("%d", id)
				} else {
					otherUserStr = "'" + otherUser.Username + "'"
				}

				terminateWithError(w, req, http.StatusForbidden, "You don't have permission to do that", fmt.Sprintf("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr))
				return
			}

			loggedOutUser, err := tqs.Logout(req.Context(), id)
			if err != nil {
				terminateWithError(w, req, http.StatusInternalServerError, "An internal server error occurred", "could not log out user: "+err.Error())
				return
			}

			var otherStr string
			if id != user.ID {
				otherStr = "user '" + loggedOutUser.Username + "'"
			} else {
				otherStr = "self"
			}

			terminateWithJSON(w, req, http.StatusNoContent, nil, fmt.Sprintf("user '%s' successfully logged out %s", user.Username, otherStr))
			return
		} else {
			terminateWithError(w, req, http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
			return
		}
	}
}

// Login returns the JWT token after logging in.
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

func (tqs TunaQuestServer) Logout(ctx context.Context, who uuid.UUID) (dao.User, error) {
	existing, err := tqs.db.Users.GetByID(ctx, who)
	if err != nil {
		return dao.User{}, fmt.Errorf("could not retrieve user")
	}

	existing.LastLogoutTime = time.Now()

	updated, err := tqs.db.Users.Update(ctx, existing)
	if err != nil {
		return dao.User{}, fmt.Errorf("could not update user")
	}

	return updated, nil
}

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

// if status is http.StatusNoContent, respObj will not be read and may be nil. Otherwise, respObj MUST NOT be nil.
func terminateWithJSON(w http.ResponseWriter, req *http.Request, status int, respObj interface{}, internalMsg string) {
	var respJSON []byte
	if status != http.StatusNoContent {
		var err error
		respJSON, err = json.Marshal(respObj)
		if err != nil {
			terminateWithError(w, req, status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
			return
		}
	}

	logHttpResponse("INFO", req, status, internalMsg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if status != http.StatusNoContent {
		w.Write(respJSON)
	}
}

func terminateWithError(w http.ResponseWriter, req *http.Request, status int, userMsg, internalMsg string) {
	logHttpResponse("ERROR", req, status, internalMsg)
	http.Error(w, userMsg, status)
}

func logHttpResponse(level string, req *http.Request, respStatus int, msg string) {
	if len(level) > 5 {
		level = level[0:5]
	}

	for len(level) < 5 {
		level += " "
	}

	log.Printf("%s: %s %s: HTTP-%d: %s", level, req.Method, req.URL.Path, respStatus, msg)
}

func (tqs TunaQuestServer) requireJWT(ctx context.Context, req *http.Request) (dao.User, error) {
	var user dao.User

	tok, err := getJWT(req)
	if err != nil {
		return dao.User{}, err
	}

	_, err = jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		// who is the user? we need this for further verification
		subj, err := t.Claims.GetSubject()
		if err != nil {
			return nil, fmt.Errorf("cannot get subject: %w", err)
		}

		id, err := uuid.Parse(subj)
		if err != nil {
			return nil, fmt.Errorf("cannot parse subject UUID: %w", err)
		}

		user, err = tqs.db.Users.GetByID(ctx, id)
		if err != nil {
			if err == inmem.ErrNotFound {
				return nil, fmt.Errorf("subject does not exist")
			} else {
				return nil, fmt.Errorf("subject could not be validated")
			}
		}

		var signKey []byte
		signKey = append(signKey, fakeTestKey...)
		signKey = append(signKey, []byte(user.Password)...)
		signKey = append(signKey, []byte(fmt.Sprintf("%d", user.LastLogoutTime.Unix()))...)
		return signKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}), jwt.WithIssuer("tqs"), jwt.WithLeeway(time.Minute))

	if err != nil {
		return dao.User{}, err
	}

	return user, nil
}

func getJWT(req *http.Request) (string, error) {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))

	if authHeader == "" {
		return "", fmt.Errorf("no authorization header present")
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	scheme := strings.TrimSpace(strings.ToLower(authParts[0]))
	token := strings.TrimSpace(authParts[1])

	if scheme != "bearer" {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	return token, nil
}

func generateJWTForUser(u dao.User) (string, error) {
	claims := &jwt.MapClaims{
		"iss":        "tqs",
		"exp":        time.Now().Add(time.Hour).Unix(),
		"sub":        u.ID.String(),
		"authorized": true,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	var signKey []byte
	signKey = append(signKey, fakeTestKey...)
	signKey = append(signKey, []byte(u.Password)...)
	signKey = append(signKey, []byte(fmt.Sprintf("%d", u.LastLogoutTime.Unix()))...)

	tokStr, err := tok.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return tokStr, nil
}

// v must be a pointer to a type.
func parseJSON(req *http.Request, v interface{}) error {
	contentType := req.Header.Get("Content-Type")

	if strings.ToLower(contentType) != "application/json" {
		return fmt.Errorf("request content-type is not application/json")
	}

	bodyData, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	err = json.Unmarshal(bodyData, v)
	if err != nil {
		return fmt.Errorf("malformed JSON in request")
	}

	return nil
}
