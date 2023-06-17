package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func (tqs TunaQuestServer) doEndpointLoginPOST(req *http.Request) endpointResult {
	loginData := LoginRequest{}
	err := parseJSON(req, &loginData)
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error(), err.Error())
	}

	user, err := tqs.Login(req.Context(), loginData.User, loginData.Password)
	if err != nil {
		if err == ErrBadCredentials {
			return jsonErr(http.StatusUnauthorized, err.Error(), err.Error()).
				withHeader("WWW-Authenticate", `Basic realm="TunaQuest server", charset="utf-8"`)
		} else {
			return jsonErr(http.StatusInternalServerError, "An internal server error occurred", err.Error())
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := tqs.generateJWT(user)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, "An internal server error occurred", "could not generate JWT: "+err.Error())
	}

	resp := LoginResponse{Token: tok}
	return jsonResponse(http.StatusCreated, resp, "user '"+user.Username+"' successfully logged in")
}

func (tqs TunaQuestServer) doEndpointLoginDELETE(req *http.Request, id uuid.UUID) endpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		return jsonErr(http.StatusUnauthorized, "Valid bearer JWT token required", fmt.Sprintf("could not verify JWT: %s", err.Error()))
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

		return jsonErr(http.StatusForbidden, "You don't have permission to do that", fmt.Sprintf("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr))
	}

	loggedOutUser, err := tqs.Logout(req.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			return jsonErr(http.StatusNotFound, "The requested resource was not found", "not found")
		}
		return jsonErr(http.StatusInternalServerError, "An internal server error occurred", "could not log out user: "+err.Error())
	}

	var otherStr string
	if id != user.ID {
		otherStr = "user '" + loggedOutUser.Username + "'"
	} else {
		otherStr = "self"
	}

	return jsonResponse(http.StatusNoContent, nil, fmt.Sprintf("user '%s' successfully logged out %s", user.Username, otherStr))
}

type endpointResult struct {
	isErr       bool
	isJSON      bool
	status      int
	internalMsg string
	resp        interface{}
	hdrs        [][2]string
}

func (r endpointResult) withHeader(name, val string) endpointResult {
	erCopy := endpointResult{
		isErr:       r.isErr,
		isJSON:      r.isJSON,
		status:      r.status,
		internalMsg: r.internalMsg,
		resp:        r.resp,
		hdrs:        r.hdrs,
	}

	erCopy.hdrs = append(erCopy.hdrs, [2]string{name, val})
	return erCopy
}

func (r endpointResult) writeResponse(w http.ResponseWriter, req *http.Request) {
	// if this hasn't been properly created, output error directly and do not
	// try to read properties
	if r.status == 0 {
		logHttpResponse("ERROR", req, http.StatusInternalServerError, "endpoint result was never populated")
		http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		return
	}

	var respJSON []byte
	if r.isJSON && r.status != http.StatusNoContent {
		var err error
		respJSON, err = json.Marshal(r.resp)
		if err != nil {
			res := jsonErr(r.status, "An internal server error occurred", "could not marshal JSON response: "+err.Error())
			res.writeResponse(w, req)
		}
	}

	if r.isErr {
		logHttpResponse("ERROR", req, r.status, r.internalMsg)
	} else {
		logHttpResponse("INFO", req, r.status, r.internalMsg)
	}

	var respBytes []byte

	if r.isJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		respBytes = respJSON
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.status != http.StatusNoContent {
			respBytes = []byte(fmt.Sprintf("%v", r.resp))
		}
	}

	for i := range r.hdrs {
		w.Header().Set(r.hdrs[i][0], r.hdrs[i][1])
	}

	w.WriteHeader(r.status)

	if r.status != http.StatusNoContent {
		w.Write(respBytes)
	}
}

// if status is http.StatusNoContent, respObj will not be read and may be nil.
// Otherwise, respObj MUST NOT be nil.
func jsonResponse(status int, respObj interface{}, internalMsg string) endpointResult {
	return endpointResult{
		isJSON:      true,
		isErr:       false,
		status:      status,
		internalMsg: internalMsg,
		resp:        respObj,
	}
}

func jsonErr(status int, userMsg, internalMsg string) endpointResult {
	return endpointResult{
		isJSON:      true,
		isErr:       true,
		status:      status,
		internalMsg: internalMsg,
		resp: ErrorResponse{
			Error:  userMsg,
			Status: status,
		},
	}
}

// textErr is like jsonErr but it avoids JSON encoding of any kind and writes
// the output as plain text.
func textErr(status int, userMsg, internalMsg string) endpointResult {
	return endpointResult{
		isJSON:      false,
		isErr:       true,
		status:      status,
		internalMsg: internalMsg,
		resp:        userMsg,
	}
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
