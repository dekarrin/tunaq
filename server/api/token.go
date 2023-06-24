package api

import (
	"net/http"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/middle"
	"github.com/dekarrin/tunaq/server/result"
	"github.com/dekarrin/tunaq/server/token"
)

// HTTPCreateToken returns a HandlerFunc that creates a new token for the user
// the client is logged in as.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// the logged-in user of the client making the request.
func (api API) HTTPCreateToken() http.HandlerFunc {
	return httpEndpoint(api.UnauthDelay, api.epCreateToken)
}

func (api API) epCreateToken(req *http.Request) result.Result {
	user := req.Context().Value(middle.AuthUser).(dao.User)

	tok, err := token.Generate(api.Secret, user)
	if err != nil {
		return result.InternalServerError("could not generate JWT: " + err.Error())
	}

	resp := LoginResponse{
		Token:  tok,
		UserID: user.ID.String(),
	}
	return result.Created(resp, "user '"+user.Username+"' successfully created new token")
}
