package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

type LoginResponse struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

type result struct {
	isErr  bool
	isJSON bool

	resp        interface{}
	status      int
	internalMsg string
}

func (r result) write(w http.ResponseWriter, req *http.Request) {
	// writeRaw
	if r.isErr {
		var respJSON []byte
		if r.isJSON {
			var err error
			respJSON, err = json.Marshal(r.resp)
			if err != nil {
				respondRawErr(w, req, r.status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
				return
			}
		}

		logHttpResponse("ERROR", req, r.status, r.internalMsg)

		if !r.isJSON {
			userMsg := fmt.Sprintf("%v", r.resp)
			http.Error(w, userMsg, r.status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(respJSON)
		return
	}

	// not an error, normal response.
	var respJSON []byte
	if r.isJSON && r.status != http.StatusNoContent {
		var err error
		respJSON, err = json.Marshal(r.resp)
		if err != nil {
			respondErr(w, req, r.status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
			return
		}
	}

	logHttpResponse("INFO", req, r.status, r.internalMsg)

	if r.isJSON {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(r.status)

	if r.status != http.StatusNoContent {
		if r.isJSON {
			w.Write(respJSON)
		}
	}
}

const (
	EntityLogin = "login"
)

func (tqs *TunaQuestServer) initHandlers() {
	tqs.srv.HandleFunc("/", tqs.handlePathRoot)
	tqs.srv.HandleFunc("/"+EntityLogin, tqs.handlePathLogin)
	tqs.srv.HandleFunc("/"+EntityLogin+"/", tqs.handlePathLogin)
}

func (tqs TunaQuestServer) handlePathRoot(w http.ResponseWriter, req *http.Request) {
	respondErr(w, req, http.StatusNotFound, "The requested resource was not found", "not found")
}

func (tqs TunaQuestServer) handlePathLogin(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)

	if req.URL.Path == "/"+EntityLogin+"/" || req.URL.Path == "/"+EntityLogin {
		if req.Method == http.MethodPost {
			tqs.doLoginPOST(w, req)
		} else {
			respondErr(w, req, http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
			return
		}
	} else {
		// check for /login/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			respondErr(w, req, http.StatusNotFound, "The requested resource was not found", "not found")
			return
		}

		id, err := uuid.Parse(pathParts[1])
		if err != nil {
			respondErr(w, req, http.StatusNotFound, "The requested resource was not found", "not found")
			return
		}

		if req.Method == http.MethodDelete {
			tqs.doLoginDELETE(w, req, id)
		} else {
			respondErr(w, req, http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
			return
		}
	}
}

func (tqs TunaQuestServer) doLoginPOST(w http.ResponseWriter, req *http.Request) {
	loginData := LoginRequest{}
	err := parseJSON(req, &loginData)
	if err != nil {
		respondErr(w, req, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	user, err := tqs.Login(req.Context(), loginData.User, loginData.Password)
	if err != nil {
		if err == ErrBadCredentials {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"TunaQuest server\", charset=\"utf-8\"")
			respondErr(w, req, http.StatusUnauthorized, err.Error(), err.Error())
			return
		} else {
			respondErr(w, req, http.StatusInternalServerError, "An internal server error occurred", err.Error())
			return
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := tqs.generateJWT(user)
	if err != nil {
		respondErr(w, req, http.StatusInternalServerError, "An internal server error occurred", "could not generate JWT: "+err.Error())
		return
	}

	resp := LoginResponse{Token: tok}
	respond(w, req, http.StatusCreated, resp, "user '"+user.Username+"' successfully logged in")
}

func (tqs TunaQuestServer) doLoginDELETE(w http.ResponseWriter, req *http.Request, id uuid.UUID) {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		respondErr(w, req, http.StatusUnauthorized, "Valid bearer JWT token required", fmt.Sprintf("could not verify JWT: %s", err.Error()))
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

		respondErr(w, req, http.StatusForbidden, "You don't have permission to do that", fmt.Sprintf("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr))
		return
	}

	loggedOutUser, err := tqs.Logout(req.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			respondErr(w, req, http.StatusNotFound, "The requested resource was not found", "not found")
			return
		}
		respondErr(w, req, http.StatusInternalServerError, "An internal server error occurred", "could not log out user: "+err.Error())
		return
	}

	var otherStr string
	if id != user.ID {
		otherStr = "user '" + loggedOutUser.Username + "'"
	} else {
		otherStr = "self"
	}

	respond(w, req, http.StatusNoContent, nil, fmt.Sprintf("user '%s' successfully logged out %s", user.Username, otherStr))
}

// if status is http.StatusNoContent, respObj will not be read and may be nil.
// Otherwise, respObj MUST NOT be nil.
func respond(w http.ResponseWriter, req *http.Request, status int, respObj interface{}, internalMsg string) {
	var respJSON []byte
	if status != http.StatusNoContent {
		var err error
		respJSON, err = json.Marshal(respObj)
		if err != nil {
			respondErr(w, req, status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
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

func respondErr(w http.ResponseWriter, req *http.Request, status int, userMsg, internalMsg string) {
	respErr := ErrorResponse{
		Error:  userMsg,
		Status: status,
	}
	respJSON, err := json.Marshal(respErr)
	if err != nil {
		respondRawErr(w, req, status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
		return
	}

	logHttpResponse("ERROR", req, status, internalMsg)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(respJSON)
}

// respondRawErr is like respondErr but it avoids JSON encoding
// of any kind and writes the output as plain text.
func respondRawErr(w http.ResponseWriter, req *http.Request, status int, userMsg, internalMsg string) {
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

	log.Printf("%s: HTTP-%d resonse to %s %s: %s", level, respStatus, req.Method, req.URL.Path, msg)
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

func panicTo500(w http.ResponseWriter, req *http.Request) {
	if panicErr := recover(); panicErr != nil {
		respondRawErr(
			w, req, http.StatusInternalServerError,
			"An internal server error occurred",
			fmt.Sprintf("panic: %v\n%s", panicErr, string(debug.Stack())),
		)
	}
}
