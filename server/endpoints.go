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

	"github.com/dekarrin/tunaq/internal/version"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/dekarrin/tunaq/server/tunas"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EndpointFunc func(req *http.Request) EndpointResult

func Endpoint(ep EndpointFunc) http.HandlerFunc {
	unauthTimeout := time.Second

	return func(w http.ResponseWriter, req *http.Request) {
		defer panicTo500(w, req)
		result := ep(req)

		if result.status == http.StatusUnauthorized || result.status == http.StatusForbidden || result.status == http.StatusInternalServerError {
			// if it's one of these statusus, either the user is improperly
			// logging in or tried to access a forbidden resource, both of which
			// should force the wait time before responding.
			time.Sleep(unauthTimeout)
		}

		// TODO: pull logging out of writeResponse and do that immediately after
		// getting the result so it isn't subject to the wait time.
		result.writeResponse(w, req)
	}
}

// requireIDParam gets the ID of the main entity being referenced in the URI and
// returns it. It panics if the key is not there or is not parsable.
func requireIDParam(r *http.Request) uuid.UUID {
	id, err := getURLParam(r, URLParamKeyID, uuid.Parse)
	if err != nil {
		panic(err.Error())
	}
	return id
}

func getURLParam[E any](r *http.Request, key string, parse func(string) (E, error)) (val E, err error) {
	valStr := chi.URLParam(r, key)
	if valStr == "" {
		// either it does not exist or it is nil; treat both as the same and
		// return an error
		return val, fmt.Errorf("parameter does not exist")
	}

	val, err = parse(valStr)
	if err != nil {
		return val, serr.New("", serr.ErrBadArgument)
	}
	return val, nil
}

// API holds parameters for endpoints needed to run and a service layer that
// will perform most of the actual logic. To use API, create one and then
// assign the result of its HTTP* methods as handlers to a router or some other
// kind of server mux.
//
// This is exclusively an API for serving external requests. For direct
// programmatic access into the backend of a TunaQuest server via Go code, see
// [tunas.Service].
type API struct {
	// Backend is the service that the API calls to perform the requested
	// actions.
	Backend tunas.Service

	// UnauthDelay is the amount of time that a request will pause before
	// responding with an HTTP-403, HTTP-401, or HTTP-500 to deprioritize such
	// requests from processing and I/O.
	UnauthDelay time.Duration

	// Secret is the secret used to sign JWT tokens.
	Secret []byte
}

// HTTPCreateLogin returns a HandlerFunc that uses the API to log in a user with
// a username and password and return the auth token for that user.
func (api API) HTTPCreateLogin() http.HandlerFunc {
	return Endpoint(api.epCreateLogin)
}

func (api API) epCreateLogin(req *http.Request) EndpointResult {
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

	user, err := api.Backend.Login(req.Context(), loginData.Username, loginData.Password)
	if err != nil {
		if errors.Is(err, serr.ErrBadCredentials) {
			return jsonUnauthorized(serr.ErrBadCredentials.Error(), "user '%s': %s", loginData.Username, err.Error())
		} else {
			return jsonInternalServerError(err.Error())
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := generateJWT(api.Secret, user)
	if err != nil {
		return jsonInternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return jsonCreated(resp, "user '"+user.Username+"' successfully logged in")
}

// HTTPDeleteLogin returns a HandlerFunc that deletes active login for some
// user. Only admin users can delete logins for users other themselves.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user to log out and the logged-in user of the client making the
// request.
func (api API) HTTPDeleteLogin() http.HandlerFunc {
	return Endpoint(api.epDeleteLogin)
}

func (api API) epDeleteLogin(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else's login? they'd betta be the
	// admin if so!
	if id != user.ID && user.Role != dao.Admin {
		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			if !errors.Is(err, serr.ErrNotFound) {
				return jsonInternalServerError("retrieve user for perm checking: %s", err.Error())
			}
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	loggedOutUser, err := api.Backend.Logout(req.Context(), id)
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

// HTTPCreateToken returns a HandlerFunc that creates a new token for the user
// the client is logged in as.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPCreateToken() http.HandlerFunc {
	return Endpoint(api.epCreateToken)
}

func (api API) epCreateToken(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	tok, err := generateJWT(api.Secret, user)
	if err != nil {
		return jsonInternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return jsonCreated(resp, "user '"+user.Username+"' successfully created new token")
}

// HTTPGetAllUsers returns a HandlerFunc that retrieves all existing users. Only
// an admin user can call this endpoint.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPGetAllUsers() http.HandlerFunc {
	return Endpoint(api.epGetAllUsers)
}

// GET /users: get all users (admin auth required).
func (api API) epGetAllUsers(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return jsonForbidden("user '%s' (role %s): forbidden", user.Username, user.Role)
	}

	users, err := api.Backend.GetAllUsers(req.Context())
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

// HTTPCreateUser returns a HandlerFunc that creates a new user entity. Only an
// admin user can directly create new users.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPCreateUser() http.HandlerFunc {
	return Endpoint(api.epCreateUser)
}

func (api API) epCreateUser(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
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

	newUser, err := api.Backend.CreateUser(req.Context(), createUser.Username, createUser.Password, createUser.Email, role)
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

// HTTPGetUser returns a HandlerFunc that gets an existing user. All users may
// retrieve themselves, but only an admin user can retrieve details on other
// users.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user being operated on and the logged-in user of the client
// making the request.
func (api API) HTTPGetUser() http.HandlerFunc {
	return Endpoint(api.epGetUser)
}

func (api API) epGetUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {

		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) get user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	userInfo, err := api.Backend.GetUser(req.Context(), id.String())
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

// HTTPUpdateUser returns a HandlerFunc that updates an existing user. Only
// updates to properties that are not auto-calculated are respected (e.g. trying
// to update the created time will have no effect). All users may update
// themselves, but only the admin user may update other users.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user being operated on and the logged-in user of the client
// making the request.
func (api API) HTTPUpdateUser() http.HandlerFunc {
	return Endpoint(api.epUpdateUser)
}

func (api API) epUpdateUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	if id != user.ID && user.Role != dao.Admin {
		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) update user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	var updateReq UserUpdateRequest
	err := parseJSON(req, &updateReq)
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

	existing, err := api.Backend.GetUser(req.Context(), id.String())
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
	updated, err := api.Backend.UpdateUser(req.Context(), id.String(), newID, newUsername, newEmail, newRole)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict(err.Error(), err.Error())
		} else if errors.Is(err, serr.ErrNotFound) {
			return jsonNotFound()
		}
		return jsonInternalServerError(err.Error())
	}
	if updateReq.Password.Update {
		updated, err = api.Backend.UpdatePassword(req.Context(), updated.ID.String(), updateReq.Password.Value)
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

// HTTPReplaceUser returns a HandlerFunc that replaces a user entity with a
// completely new one with the same ID. Only an admin user may replace a user.
// If the user with the given ID does not exist, it will be created.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user being replaced and the logged-in user of the client making
// the request.
func (api API) HTTPReplaceUser() http.HandlerFunc {
	return Endpoint(api.epReplaceUser)
}

func (api API) epReplaceUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
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

	newUser, err := api.Backend.CreateUser(req.Context(), createUser.Username, createUser.Password, createUser.Email, role)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return jsonConflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return jsonBadRequest(err.Error(), err.Error())
		}
		return jsonInternalServerError(err.Error())
	}

	// but also update it immediately to set its user ID
	newUser, err = api.Backend.UpdateUser(req.Context(), newUser.ID.String(), createUser.ID, newUser.Username, newUser.Email.Address, newUser.Role)
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

// HTTPDeleteUser returns a HandlerFunc that deletes a user entity. All users
// may delete themselves, but only an admin user may delete another user.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user being deleted and the logged-in user of the client making
// the request.
func (api API) HTTPDeleteUser() http.HandlerFunc {
	return Endpoint(api.epDeleteUser)
}

func (api API) epDeleteUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {
		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return jsonForbidden("user '%s' (role %s) delete user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	deletedUser, err := api.Backend.DeleteUser(req.Context(), id.String())
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

// HTTPGetInfo returns a HandlerFunc that retrieves information on the API and
// server.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// a value denoting whether the client making the request is logged-in.
func (api API) HTTPGetInfo() http.HandlerFunc {
	return Endpoint(api.epGetInfo)
}

func (api API) epGetInfo(req *http.Request) EndpointResult {
	loggedIn := req.Context().Value(AuthLoggedIn).(bool)

	var resp InfoModel
	resp.Version.Server = version.ServerCurrent
	resp.Version.TunaQuest = version.Current

	userStr := "unauthed client"
	if loggedIn {
		user := req.Context().Value(AuthUser).(dao.User)
		userStr = "user '" + user.Username + "'"
	}
	return jsonOK(resp, "%s got API info", userStr)
}

// -----------------------------------
// OLD SERVER BELOW
// -----------------------------------

// POST /login: create a new login with token
func (tqs TunaQuestServer) epCreateLogin(req *http.Request) EndpointResult {
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
		if errors.Is(err, serr.ErrBadCredentials) {
			return jsonUnauthorized(serr.ErrBadCredentials.Error(), "user '%s': %s", loginData.Username, err.Error())
		} else {
			return jsonInternalServerError(err.Error())
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := generateJWT(tqs.jwtSecret, user)
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
func (tqs TunaQuestServer) epCreateToken(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	tok, err := generateJWT(tqs.jwtSecret, user)
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
//
// TODO: move core functionality to epDeleteLogin and deprecate this once everyfin
// is fully swapped to chi router.
func (tqs TunaQuestServer) epDeleteLogin(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else's login? they'd betta be the
	// admin if so!
	if id != user.ID && user.Role != dao.Admin {
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
func (tqs TunaQuestServer) epCreateNewUser(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
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
func (tqs TunaQuestServer) epGetAllUsers(req *http.Request) EndpointResult {
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
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

func (tqs TunaQuestServer) epGetUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {

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

func (tqs TunaQuestServer) epUpdateUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	if id != user.ID && user.Role != dao.Admin {
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
	err := parseJSON(req, &updateReq)
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

func (tqs TunaQuestServer) epCreateExistingUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return jsonForbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
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

func (tqs TunaQuestServer) epDeleteUser(req *http.Request) EndpointResult {
	id := requireIDParam(req)
	user := req.Context().Value(AuthUser).(dao.User)

	// is the user trying to delete someone else? they'd betta be the admin if so!
	if id != user.ID && user.Role != dao.Admin {
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

// GET /info: get server info
func (tqs TunaQuestServer) epGetInfo(req *http.Request) EndpointResult {
	loggedIn := req.Context().Value(AuthLoggedIn).(bool)

	var resp InfoModel
	resp.Version.Server = version.ServerCurrent
	resp.Version.TunaQuest = version.Current

	userStr := "unauthed client"
	if loggedIn {
		user := req.Context().Value(AuthUser).(dao.User)
		userStr = "user '" + user.Username + "'"
	}
	return jsonOK(resp, "%s got API info", userStr)
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
