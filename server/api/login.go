package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/middle"
	"github.com/dekarrin/tunaq/server/result"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/dekarrin/tunaq/server/token"
)

// HTTPCreateLogin returns a HandlerFunc that uses the API to log in a user with
// a username and password and return the auth token for that user.
func (api API) HTTPCreateLogin() http.HandlerFunc {
	return Endpoint(api.epCreateLogin)
}

func (api API) epCreateLogin(req *http.Request) result.Result {
	loginData := LoginRequest{}
	err := parseJSON(req, &loginData)
	if err != nil {
		return result.BadRequest(err.Error(), err.Error())
	}

	if loginData.Username == "" {
		return result.BadRequest("username: property is empty or missing from request", "empty username")
	}
	if loginData.Password == "" {
		return result.BadRequest("password: property is empty or missing from request", "empty password")
	}

	user, err := api.Backend.Login(req.Context(), loginData.Username, loginData.Password)
	if err != nil {
		if errors.Is(err, serr.ErrBadCredentials) {
			return result.Unauthorized(serr.ErrBadCredentials.Error(), "user '%s': %s", loginData.Username, err.Error())
		} else {
			return result.InternalServerError(err.Error())
		}
	}

	// build the token
	// password is valid, generate token for user and return it.
	tok, err := token.Generate(api.Secret, user)
	if err != nil {
		return result.InternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return result.Created(resp, "user '"+user.Username+"' successfully logged in")
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

func (api API) epDeleteLogin(req *http.Request) result.Result {
	id := requireIDParam(req)
	user := req.Context().Value(middle.AuthUser).(dao.User)

	// is the user trying to delete someone else's login? they'd betta be the
	// admin if so!
	if id != user.ID && user.Role != dao.Admin {
		var otherUserStr string
		otherUser, err := api.Backend.GetUser(req.Context(), id.String())
		// if there was another user, find out now
		if err != nil {
			if !errors.Is(err, serr.ErrNotFound) {
				return result.InternalServerError("retrieve user for perm checking: %s", err.Error())
			}
			otherUserStr = fmt.Sprintf("%d", id)
		} else {
			otherUserStr = "'" + otherUser.Username + "'"
		}

		return result.Forbidden("user '%s' (role %s) logout of user %s: forbidden", user.Username, user.Role, otherUserStr)
	}

	loggedOutUser, err := api.Backend.Logout(req.Context(), id)
	if err != nil {
		if errors.Is(err, serr.ErrNotFound) {
			return result.NotFound()
		}
		return result.InternalServerError("could not log out user: " + err.Error())
	}

	var otherStr string
	if id != user.ID {
		otherStr = "user '" + loggedOutUser.Username + "'"
	} else {
		otherStr = "self"
	}

	return result.NoContent("user '%s' successfully logged out %s", user.Username, otherStr)
}
