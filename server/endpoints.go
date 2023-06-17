package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

// POST /login: create a new login with token
func (tqs TunaQuestServer) doEndpoint_Login_POST(req *http.Request) endpointResult {
	loginData := LoginRequest{}
	err := parseJSON(req, &loginData)
	if err != nil {
		return jsonBadRequest(err.Error(), err.Error())
	}

	if loginData.Username == "" {
		return jsonBadRequest("username: property is empty or missing from request", "empty user")
	}
	if loginData.Password == "" {
		return jsonBadRequest("password: property is empty or missing from request", "empty password")
	}

	user, err := tqs.Login(req.Context(), loginData.Username, loginData.Password)
	if err != nil {
		if errors.Is(err, ErrBadCredentials) {
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

// POST /tokens: create a new token for self (auth required)
func (tqs TunaQuestServer) doEndpoint_Token_POST(req *http.Request) endpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		return jsonUnauthorized(err.Error())
	}

	tok, err := tqs.generateJWT(user)
	if err != nil {
		return jsonInternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return jsonCreated(resp, "user '"+user.Username+"' successfully created new token")
}

// DELETE /login/{id}: remove a login for some user (log out). Requires auth for
// access at all. Requires auth by user with role Admin to log out anybody but
// self.
func (tqs TunaQuestServer) doEndpoint_LoginID_DELETE(req *http.Request, id uuid.UUID) endpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		return jsonUnauthorized(err.Error())
	}

	// is the user trying to delete someone else's login? they'd betta be the
	// admin if so!
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
		if errors.Is(err, ErrNotFound) {
			return jsonNotFound()
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

// DELETE /users/{id}: delete a user. Requires admin auth for any but own ID.
func (tqs TunaQuestServer) doEndpoint_UsersID_DELETE(req *http.Request, id uuid.UUID) endpointResult {
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

		return jsonForbidden("user '%s' (role %s) delete user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	deletedUser, err := tqs.DeleteUser(req.Context(), id.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return jsonNotFound()
		} else if errors.Is(err, ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		}
		return jsonInternalServerError("could not delete user: " + err.Error())
	}

	var otherStr string
	if id != user.ID {
		otherStr = "user '" + deletedUser.Username + "'"
	} else {
		otherStr = "self"
	}

	return jsonNoContent("user '%s' successfully deleted %s", user.Username, otherStr)
}

// POST /users: create a new user (admin auth required)
func (tqs TunaQuestServer) doEndpoint_Users_POST(req *http.Request) endpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		return jsonUnauthorized(err.Error())
	}

	if user.Role != dao.Admin {
		return jsonForbidden()
	}

	var createUser UserModel
	err = parseJSON(req, &createUser)
	if err != nil {
		return jsonBadRequest(err.Error(), err.Error())
	}
	if createUser.Username == "" {
		return jsonBadRequest("username: property is empty or missing from request", "empty username")
	}
	if createUser.Password == "" {
		return jsonBadRequest("password: property is empty or missing from request", "empty password")
	}

	role := dao.Unverified
	if createUser.Role != "" {
		role, err = dao.ParseRole(createUser.Role)
		if err != nil {
			return jsonBadRequest("role: "+err.Error(), "role: %s", err.Error())
		}
	}

	newUser, err := tqs.CreateUser(req.Context(), createUser.Username, createUser.Password, createUser.Email, role)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return jsonConflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		} else {
			return jsonInternalServerError(err.Error())
		}
	}

	resp := UserModel{
		ID:       newUser.ID.String(),
		Username: newUser.Username,
		Role:     newUser.Role.String(),
	}

	if newUser.Email != nil {
		resp.Email = newUser.Email.String()
	}

	return jsonCreated(resp, "user '%s' (%s) created", resp.Username, resp.ID)
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
