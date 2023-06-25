package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/middle"
	"github.com/dekarrin/tunaq/server/result"
	"github.com/dekarrin/tunaq/server/serr"
)

// HTTPGetAllUsers returns a HandlerFunc that retrieves all existing users. Only
// an admin user can call this endpoint.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPGetAllUsers() http.HandlerFunc {
	return httpEndpoint(api.UnauthDelay, api.epGetAllUsers)
}

// GET /users: get all users (admin auth required).
func (api API) epGetAllUsers(req *http.Request) result.Result {
	user := req.Context().Value(middle.AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return result.Forbidden("user '%s' (role %s): forbidden", user.Username, user.Role)
	}

	users, err := api.Backend.GetAllUsers(req.Context())
	if err != nil {
		return result.InternalServerError(err.Error())
	}

	resp := make([]UserModel, len(users))

	for i := range users {
		resp[i] = UserModel{
			URI:            PathPrefix + "/users/" + users[i].ID.String(),
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

	return result.OK(resp, "user '%s' got all users", user.Username)
}

// HTTPCreateUser returns a HandlerFunc that creates a new user entity. Only an
// admin user can directly create new users.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPCreateUser() http.HandlerFunc {
	return httpEndpoint(api.UnauthDelay, api.epCreateUser)
}

func (api API) epCreateUser(req *http.Request) result.Result {
	user := req.Context().Value(middle.AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return result.Forbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
	if err != nil {
		return result.BadRequest(err.Error(), err.Error())
	}
	if createUser.Username == "" {
		return result.BadRequest("username: property is empty or missing from request", "empty username")
	}
	if createUser.Password == "" {
		return result.BadRequest("password: property is empty or missing from request", "empty password")
	}

	role := dao.Unverified
	if createUser.Role != "" {
		role, err = dao.ParseRole(createUser.Role)
		if err != nil {
			return result.BadRequest("role: "+err.Error(), "role: %s", err.Error())
		}
	}

	newUser, err := api.Backend.CreateUser(req.Context(), createUser.Username, createUser.Password, createUser.Email, role)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return result.Conflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return result.BadRequest(err.Error(), err.Error())
		} else {
			return result.InternalServerError(err.Error())
		}
	}

	resp := UserModel{
		URI:            PathPrefix + "/users/" + newUser.ID.String(),
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

	return result.Created(resp, "user '%s' (%s) created", resp.Username, resp.ID)
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
	return httpEndpoint(api.UnauthDelay, api.epGetUser)
}

func (api API) epGetUser(req *http.Request) result.Result {
	id := requireIDParam(req)
	user := req.Context().Value(middle.AuthUser).(dao.User)

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

		return result.Forbidden("user '%s' (role %s) get user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	userInfo, err := api.Backend.GetUser(req.Context(), id.String())
	if err != nil {
		if errors.Is(err, serr.ErrBadArgument) {
			return result.BadRequest(err.Error(), err.Error())
		} else if errors.Is(err, serr.ErrNotFound) {
			return result.NotFound()
		}
		return result.InternalServerError("could not get user: " + err.Error())
	}

	// put it into a model to return
	resp := UserModel{
		URI:            PathPrefix + "/users/" + userInfo.ID.String(),
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

	return result.OK(resp, "user '%s' successfully got %s", user.Username, otherStr)
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
	return httpEndpoint(api.UnauthDelay, api.epUpdateUser)
}

func (api API) epUpdateUser(req *http.Request) result.Result {
	id := requireIDParam(req)
	user := req.Context().Value(middle.AuthUser).(dao.User)

	if id != user.ID && user.Role != dao.Admin {
		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return result.Forbidden("user '%s' (role %s) update user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	var updateReq UserUpdateRequest
	err := parseJSON(req, &updateReq)
	if err != nil {
		if errors.Is(err, serr.ErrBodyUnmarshal) {
			// did they send a normal user?
			var normalUser UserModel
			err2 := parseJSON(req, &normalUser)
			if err2 == nil {
				return result.BadRequest("updated fields must be objects with keys {'u': true, 'v': NEW_VALUE}", "request is UserModel, not UserUpdateRequest")
			}
		}

		return result.BadRequest(err.Error(), err.Error())
	}

	// pre-parse updateRole if needed so we return bad request before hitting
	// DB
	var updateRole dao.Role
	if updateReq.Role.Update {
		updateRole, err = dao.ParseRole(updateReq.Role.Value)
		if err != nil {
			return result.BadRequest(err.Error(), err.Error())
		}
	}

	existing, err := api.Backend.GetUser(req.Context(), id.String())
	if err != nil {
		if errors.Is(err, serr.ErrNotFound) {
			return result.NotFound()
		}
		return result.InternalServerError(err.Error())
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
			return result.Conflict(err.Error(), err.Error())
		} else if errors.Is(err, serr.ErrNotFound) {
			return result.NotFound()
		}
		return result.InternalServerError(err.Error())
	}
	if updateReq.Password.Update {
		updated, err = api.Backend.UpdatePassword(req.Context(), updated.ID.String(), updateReq.Password.Value)
		if errors.Is(err, serr.ErrNotFound) {
			return result.NotFound()
		}
		return result.InternalServerError(err.Error())
	}

	resp := UserModel{
		URI:            PathPrefix + "/users/" + updated.ID.String(),
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

	return result.Created(resp, "user '%s' (%s) updated", resp.Username, resp.ID)
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
	return httpEndpoint(api.UnauthDelay, api.epReplaceUser)
}

func (api API) epReplaceUser(req *http.Request) result.Result {
	id := requireIDParam(req)
	user := req.Context().Value(middle.AuthUser).(dao.User)

	if user.Role != dao.Admin {
		return result.Forbidden("user '%s' (role %s) creation of new user: forbidden", user.Username, user.Role)
	}

	var createUser UserModel
	err := parseJSON(req, &createUser)
	if err != nil {
		return result.BadRequest(err.Error(), err.Error())
	}
	if createUser.Username == "" {
		return result.BadRequest("username: property is empty or missing from request", "empty username")
	}
	if createUser.Password == "" {
		return result.BadRequest("password: property is empty or missing from request", "empty password")
	}
	if createUser.ID == "" {
		createUser.ID = id.String()
	}
	if createUser.ID != id.String() {
		return result.BadRequest("id: must be same as ID in URI", "body ID different from URI ID")
	}

	role := dao.Unverified
	if createUser.Role != "" {
		role, err = dao.ParseRole(createUser.Role)
		if err != nil {
			return result.BadRequest("role: "+err.Error(), "role: %s", err.Error())
		}
	}

	newUser, err := api.Backend.CreateUser(req.Context(), createUser.Username, createUser.Password, createUser.Email, role)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return result.Conflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return result.BadRequest(err.Error(), err.Error())
		}
		return result.InternalServerError(err.Error())
	}

	// but also update it immediately to set its user ID
	newUser, err = api.Backend.UpdateUser(req.Context(), newUser.ID.String(), createUser.ID, newUser.Username, newUser.Email.Address, newUser.Role)
	if err != nil {
		if errors.Is(err, serr.ErrAlreadyExists) {
			return result.Conflict("User with that username already exists", "user '%s' already exists", createUser.Username)
		} else if errors.Is(err, serr.ErrBadArgument) {
			return result.BadRequest(err.Error(), err.Error())
		}
		return result.InternalServerError(err.Error())
	}

	resp := UserModel{
		URI:            PathPrefix + "/users/" + newUser.ID.String(),
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

	return result.Created(resp, "user '%s' (%s) created", resp.Username, resp.ID)
}

// HTTPDeleteUser returns a HandlerFunc that deletes a user entity. All users
// may delete themselves, but only an admin user may delete another user.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the ID of the user being deleted and the logged-in user of the client making
// the request.
func (api API) HTTPDeleteUser() http.HandlerFunc {
	return httpEndpoint(api.UnauthDelay, api.epDeleteUser)
}

func (api API) epDeleteUser(req *http.Request) result.Result {
	id := requireIDParam(req)
	user := req.Context().Value(middle.AuthUser).(dao.User)

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

		return result.Forbidden("user '%s' (role %s) delete user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	deletedUser, err := api.Backend.DeleteUser(req.Context(), id.String())
	if err != nil && !errors.Is(err, serr.ErrNotFound) {
		if errors.Is(err, serr.ErrBadArgument) {
			return result.BadRequest(err.Error(), err.Error())
		}
		return result.InternalServerError("could not delete user: " + err.Error())
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

	return result.NoContent("user '%s' successfully deleted %s", user.Username, otherStr)
}
