package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
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
		return jsonBadRequest(err.Error(), err.Error())
	}

	if loginData.User == "" {
		return jsonBadRequest("Non-empty 'user' property is empty or missing from request", "login request does not give user")
	}
	if loginData.Password == "" {
		return jsonBadRequest("Non-empty 'password' property is empty or missing from request", "login request does not give password")
	}

	user, err := tqs.Login(req.Context(), loginData.User, loginData.Password)
	if err != nil {
		if err == ErrBadCredentials {
			return jsonUnauthorized(err.Error())
		} else {
			return jsonInternalServerError(err.Error())
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := tqs.generateJWT(user)
	if err != nil {
		return jsonInternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return jsonCreated(resp, "user '"+user.Username+"' successfully logged in")
}

func (tqs TunaQuestServer) doEndpointLoginDELETE(req *http.Request, id uuid.UUID) endpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		return jsonUnauthorized(err.Error())
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

		return jsonForbidden("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	loggedOutUser, err := tqs.Logout(req.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			return jsonNotFound("not found")
		}
		return jsonInternalServerError("could not log out user: " + err.Error())
	}

	var otherStr string
	if id != user.ID {
		otherStr = "user '" + loggedOutUser.Username + "'"
	} else {
		otherStr = "self"
	}

	return jsonNoContent("user '%s' successfully logged out %s", user.Username, otherStr)
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
