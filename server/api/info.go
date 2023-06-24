package api

import (
	"net/http"

	"github.com/dekarrin/tunaq/internal/version"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/middle"
	"github.com/dekarrin/tunaq/server/result"
)

// HTTPGetInfo returns a HandlerFunc that retrieves information on the API and
// server.
//
// The handler has requirements for the request context it receives, and if the
// requirements are not met it may return an HTTP-500. The context must contain
// a value denoting whether the client making the request is logged-in.
func (api API) HTTPGetInfo() http.HandlerFunc {
	return Endpoint(api.epGetInfo)
}

func (api API) epGetInfo(req *http.Request) result.Result {
	loggedIn := req.Context().Value(middle.AuthLoggedIn).(bool)

	var resp InfoModel
	resp.Version.Server = version.ServerCurrent
	resp.Version.TunaQuest = version.Current

	userStr := "unauthed client"
	if loggedIn {
		user := req.Context().Value(middle.AuthUser).(dao.User)
		userStr = "user '" + user.Username + "'"
	}
	return result.OK(resp, "%s got API info", userStr)
}
