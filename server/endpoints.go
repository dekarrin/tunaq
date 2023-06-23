package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/google/uuid"
)

type EndpointKey int64

const (
	EndpointParams EndpointKey = iota
)

type endpointHandler struct {
	fn EndpointFunc
}

func Endpoint(ep EndpointFunc) http.Handler {
	return endpointHandler{fn: ep}
}

func (eh endpointHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	params := req.Context().Value(EndpointParams)
	paramsMap := params.(map[string]any)

	result = eh.fn(req, paramsMap)
}

// POST /login: create a new login with token
func (tqs TunaQuestServer) doEndpoint_Login_POST(req *http.Request) EndpointResult {
	loginData := LoginRequest{}
	err := parseJSON(req, &loginData)
	if err != nil {
		return jsonBadRequest(err.Error(), err.Error())
	}

	if loginData.Username == "" {
		return jsonBadRequest("username: property is empty or missing from request", "empty username")
	}
	if loginData.Password == "" {
		return jsonBadRequest("password: property is empty or missing from request", "empty password")
	}

	user, err := tqs.Login(req.Context(), loginData.Username, loginData.Password)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		if errors.Is(err, serr.ErrBadCredentials) {
			return jsonUnauthorized(serr.ErrBadCredentials.Error(), "user '%s': %s", loginData.Username, err.Error())
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
func (tqs TunaQuestServer) doEndpoint_Tokens_POST(req *http.Request) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
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
func (tqs TunaQuestServer) doEndpoint_LoginID_DELETE(req *http.Request, id uuid.UUID) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	// is the user trying to delete someone else's login? they'd betta be the
	// admin if so!
	if id != user.ID && user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)

		var otherUserStr string
		otherUser, err := tqs.db.Users().GetByID(req.Context(), id)
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
		if errors.Is(err, serr.ErrNotFound) {
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

// POST /users: create a new user (admin auth required)
func (tqs TunaQuestServer) doEndpoint_Users_POST(req *http.Request) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	if user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
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
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		} else {
			return jsonInternalServerError(err.Error())
		}
	}

	resp := UserModel{
		URI:            APIPathPrefix + "/users/" + newUser.ID.String(),
		ID:             newUser.ID.String(),
		Username:       newUser.Username,
		Role:           newUser.Role.String(),
		Created:        newUser.Created.Format(time.RFC3339),
		Modified:       newUser.Modified.Format(time.RFC3339),
		LastLogoutTime: newUser.LastLogoutTime.Format(time.RFC3339),
		LastLoginTime:  newUser.LastLoginTime.Format(time.RFC3339),
	}

	if newUser.Email != nil {
		resp.Email = newUser.Email.Address
	}

	return jsonCreated(resp, "user '%s' (%s) created", resp.Username, resp.ID)
}

// GET /users: get all users (admin auth required).
func (tqs TunaQuestServer) doEndpoint_Users_GET(req *http.Request) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	if user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)
		return jsonForbidden("user '%s' (role %s): forbidden", user.Username, user.Role)
	}

	users, err := tqs.GetAllUsers(req.Context())
	if err != nil {
		return jsonInternalServerError(err.Error())
	}

	resp := make([]UserModel, len(users))

	for i := range users {
		resp[i] = UserModel{
			URI:            APIPathPrefix + "/users/" + users[i].ID.String(),
			ID:             users[i].ID.String(),
			Username:       users[i].Username,
			Role:           users[i].Role.String(),
			Created:        users[i].Created.Format(time.RFC3339),
			Modified:       users[i].Modified.Format(time.RFC3339),
			LastLogoutTime: users[i].LastLogoutTime.Format(time.RFC3339),
			LastLoginTime:  users[i].LastLoginTime.Format(time.RFC3339),
		}
		if users[i].Email != nil {
			resp[i].Email = users[i].Email.Address
		}
	}

	return jsonOK(resp, "user '%s' got all users", user.Username)
}

// GET /users/{id}: get info on a user. Requires auth. Requires admin auth for
// any but own ID.
func (tqs TunaQuestServer) doEndpoint_UsersID_GET(req *http.Request, id uuid.UUID) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)

		var otherUserStr string
		otherUser, err := tqs.db.Users().GetByID(req.Context(), id)
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) get user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	userInfo, err := tqs.GetUser(req.Context(), id.String())
	if err != nil {
		if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		} else if errors.Is(err, serr.ErrNotFound) {
			return jsonNotFound()
		}
		return jsonInternalServerError("could not get user: " + err.Error())
	}

	// put it into a model to return
	resp := UserModel{
		URI:            APIPathPrefix + "/users/" + userInfo.ID.String(),
		ID:             userInfo.ID.String(),
		Username:       userInfo.Username,
		Role:           userInfo.Role.String(),
		Created:        userInfo.Created.Format(time.RFC3339),
		Modified:       userInfo.Modified.Format(time.RFC3339),
		LastLogoutTime: userInfo.LastLogoutTime.Format(time.RFC3339),
		LastLoginTime:  userInfo.LastLoginTime.Format(time.RFC3339),
	}
	if userInfo.Email != nil {
		resp.Email = userInfo.Email.Address
	}

	var otherStr string
	if id != user.ID {
		if userInfo.Username != "" {
			otherStr = "user '" + userInfo.Username + "'"
		} else {
			otherStr = "user " + id.String() + " (no-op)"
		}
	} else {
		otherStr = "self"
	}

	return jsonOK(resp, "user '%s' successfully got %s", user.Username, otherStr)
}

// PATCH /users/{id}: perform a partial update on an existing user with the
// given ID. Auth required. Admin auth required for modifying someone else's
// user.
func (tqs TunaQuestServer) doEndpoint_UsersID_PATCH(req *http.Request, id uuid.UUID) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	if id != user.ID && user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)

		var otherUserStr string
		otherUser, err := tqs.db.Users().GetByID(req.Context(), id)
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) update user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	var updateReq UserUpdateRequest
	err = parseJSON(req, &updateReq)
	if err != nil {
		if errors.Is(err, serr.ErrBodyUnmarshal) {
			// did they send a normal user?
			var normalUser UserModel
			err2 := parseJSON(req, &normalUser)
			if err2 == nil {
				return jsonBadRequest("updated fields must be objects with keys {'u': true, 'v': NEW_VALUE}", "request is UserModel, not UserUpdateRequest")
			}
		}

		return jsonBadRequest(err.Error(), err.Error())
	}

	// pre-parse updateRole if needed so we return bad request before hitting
	// DB
	var updateRole dao.Role
	if updateReq.Role.Update {
		updateRole, err = dao.ParseRole(updateReq.Role.Value)
		if err != nil {
			return jsonBadRequest(err.Error(), err.Error())
		}
	}

	existing, err := tqs.GetUser(req.Context(), id.String())
	if err != nil {
		if errors.Is(err, serr.ErrNotFound) {
			return jsonNotFound()
		}
		return jsonInternalServerError(err.Error())
	}

	var newEmail string
	if existing.Email != nil {
		newEmail = existing.Email.Address
	}
	if updateReq.Email.Update {
		newEmail = updateReq.Email.Value
	}
	newID := existing.ID.String()
	if updateReq.ID.Update {
		newID = updateReq.ID.Value
	}
	newUsername := existing.Username
	if updateReq.Username.Update {
		newUsername = updateReq.Username.Value
	}
	newRole := existing.Role
	if updateReq.Role.Update {
		newRole = updateRole
	}

	// TODO: this is sequential modification. we need to update this when we get
	// transactions on dao.
	updated, err := tqs.UpdateUser(req.Context(), id.String(), newID, newUsername, newEmail, newRole)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict(err.Error(), err.Error())
		} else if errors.Is(err, serr.ErrNotFound) {
			return jsonNotFound()
		}
		return jsonInternalServerError(err.Error())
	}
	if updateReq.Password.Update {
		updated, err = tqs.UpdatePassword(req.Context(), updated.ID.String(), updateReq.Password.Value)
		if errors.Is(err, serr.ErrNotFound) {
			return jsonNotFound()
		}
		return jsonInternalServerError(err.Error())
	}

	resp := UserModel{
		URI:            APIPathPrefix + "/users/" + updated.ID.String(),
		ID:             updated.ID.String(),
		Username:       updated.Username,
		Role:           updated.Role.String(),
		Created:        updated.Created.Format(time.RFC3339),
		Modified:       updated.Modified.Format(time.RFC3339),
		LastLogoutTime: updated.LastLogoutTime.Format(time.RFC3339),
		LastLoginTime:  updated.LastLoginTime.Format(time.RFC3339),
	}

	if updated.Email != nil {
		resp.Email = updated.Email.Address
	}

	return jsonCreated(resp, "user '%s' (%s) updated", resp.Username, resp.ID)
}

// PUT /users/{id}: create an existing user with the given ID (admin auth
// required)
func (tqs TunaQuestServer) doEndpoint_UsersID_PUT(req *http.Request, id uuid.UUID) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	if user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
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
	if createUser.ID == "" {
		createUser.ID = id.String()
	}
	if createUser.ID != id.String() {
		return jsonBadRequest("id: must be same as ID in URI", "body ID different from URI ID")
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
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		}
		return jsonInternalServerError(err.Error())
	}

	// but also update it immediately to set its user ID
	newUser, err = tqs.UpdateUser(req.Context(), newUser.ID.String(), createUser.ID, newUser.Username, newUser.Email.Address, newUser.Role)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		}
		return jsonInternalServerError(err.Error())
	}

	resp := UserModel{
		URI:            APIPathPrefix + "/users/" + newUser.ID.String(),
		ID:             newUser.ID.String(),
		Username:       newUser.Username,
		Role:           newUser.Role.String(),
		Created:        newUser.Created.Format(time.RFC3339),
		Modified:       newUser.Modified.Format(time.RFC3339),
		LastLogoutTime: newUser.LastLogoutTime.Format(time.RFC3339),
		LastLoginTime:  newUser.LastLoginTime.Format(time.RFC3339),
	}

	if newUser.Email != nil {
		resp.Email = newUser.Email.Address
	}

	return jsonCreated(resp, "user '%s' (%s) created", resp.Username, resp.ID)
}

// DELETE /users/{id}: delete a user. Requires auth. Requires admin auth for any
// but own ID.
func (tqs TunaQuestServer) doEndpoint_UsersID_DELETE(req *http.Request, id uuid.UUID) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)

		var otherUserStr string
		otherUser, err := tqs.db.Users().GetByID(req.Context(), id)
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) delete user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	deletedUser, err := tqs.DeleteUser(req.Context(), id.String())
	if err != nil && !errors.Is(err, serr.ErrNotFound) {
		if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		}
		return jsonInternalServerError("could not delete user: " + err.Error())
	}

	var otherStr string
	if id != user.ID {
		if deletedUser.Username != "" {
			otherStr = "user '" + deletedUser.Username + "'"
		} else {
			otherStr = "user " + id.String() + " (no-op)"
		}
	} else {
		otherStr = "self"
	}

	return jsonNoContent("user '%s' successfully deleted %s", user.Username, otherStr)
}

// GET /users: get all users (admin auth required).
func (tqs TunaQuestServer) doEndpoint_Info_GET(req *http.Request) EndpointResult {
	user, err := tqs.requireJWT(req.Context(), req)
	if err != nil {
		time.Sleep(tqs.unauthedDelay)
		return jsonUnauthorized("", err.Error())
	}

	if user.Role != dao.Admin {
		time.Sleep(tqs.unauthedDelay)
		return jsonForbidden("user '%s' (role %s): forbidden", user.Username, user.Role)
	}

	users, err := tqs.GetAllUsers(req.Context())
	if err != nil {
		return jsonInternalServerError(err.Error())
	}

	resp := make([]UserModel, len(users))

	for i := range users {
		resp[i] = UserModel{
			URI:            APIPathPrefix + "/users/" + users[i].ID.String(),
			ID:             users[i].ID.String(),
			Username:       users[i].Username,
			Role:           users[i].Role.String(),
			Created:        users[i].Created.Format(time.RFC3339),
			Modified:       users[i].Modified.Format(time.RFC3339),
			LastLogoutTime: users[i].LastLogoutTime.Format(time.RFC3339),
			LastLoginTime:  users[i].LastLoginTime.Format(time.RFC3339),
		}
		if users[i].Email != nil {
			resp[i].Email = users[i].Email.Address
		}
	}

	return jsonOK(resp, "user '%s' got all users", user.Username)
}

// v must be a pointer to a type. Will return error such that
// errors.Is(err, ErrMalformedBody) returns true if it is problem decoding the
// JSON itself.
func parseJSON(req *http.Request, v interface{}) error {
	contentType := req.Header.Get("Content-Type")

	if strings.ToLower(contentType) != "application/json" {
		return fmt.Errorf("request content-type is not application/json")
	}

	bodyData, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}
	defer func() {
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyData))
	}()

	err = json.Unmarshal(bodyData, v)
	if err != nil {
		return serr.New("malformed JSON in request", err, serr.ErrBodyUnmarshal)
	}

	return nil
}
