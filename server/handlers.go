package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/go-chi/chi/v5"
)

const (
	APIPathPrefix = "/api/v1"
)

var (
	paramTypePats = map[string]string{
		"uuid": "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}",
	}
)

// p is a quick parameter in a URI, made very small to ease readability in route
// listings.
func p(nameType string) string {
	var name string
	var pat string

	parts := strings.SplitN(nameType, ":", 2)
	name = parts[0]
	if len(parts) == 2 {
		// we have a type, if it's a name in the paramTypePats map use that else
		// treat it as a normal pattern
		pat = parts[1]

		if translatedPat, ok := paramTypePats[parts[1]]; ok {
			pat = translatedPat
		}
	}

	if pat == "" {
		return "{" + name + "}"
	}
	return "{" + name + ":" + pat + "}"
}

func newRouter(api API) chi.Router {
	r := chi.NewRouter()

	r.Mount(APIPathPrefix, newAPIRouter(api))

	return r
}

func newAPIRouter(api API) chi.Router {
	r := chi.NewRouter()

	login := newLoginRouter(api)
	tokens := newTokensRouter(api)
	users := newUsersRouter(api)
	info := newInfoRouter(api)

	r.Mount("/login", login)
	r.Mount("/tokens", tokens)
	r.Mount("/users", users)
	r.Mount("/info", info)
	r.HandleFunc("/info/", RedirectNoTrailingSlash)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		jsonNotFound().writeResponse(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(api.UnauthDelay)
		jsonMethodNotAllowed(r).writeResponse(w, r)
	})

	return r
}

func newLoginRouter(api API) chi.Router {
	reqAuth := RequireAuth(api.Backend.DB.Users(), api.Secret, api.UnauthDelay, dao.User{})

	r := chi.NewRouter()

	r.Post("/", api.HTTPCreateLogin())
	r.With(reqAuth).Delete("/"+p("id:uuid"), api.HTTPDeleteLogin())
	r.HandleFunc("/"+p("id:uuid")+"/", RedirectNoTrailingSlash)

	return r
}

func newTokensRouter(api API) chi.Router {
	reqAuth := RequireAuth(api.Backend.DB.Users(), api.Secret, api.UnauthDelay, dao.User{})

	r := chi.NewRouter()

	r.With(reqAuth).Post("/", api.HTTPCreateToken())

	return r
}

func newUsersRouter(api API) chi.Router {
	reqAuth := RequireAuth(api.Backend.DB.Users(), api.Secret, api.UnauthDelay, dao.User{})

	r := chi.NewRouter()

	r.Use(reqAuth)

	r.Get("/", api.HTTPGetAllUsers())
	r.Post("/", api.HTTPCreateUser())

	r.Route("/"+p("id:uuid"), func(r chi.Router) {
		r.Get("/", api.HTTPGetUser())
		r.Put("/", api.HTTPReplaceUser())
		r.Patch("/", api.HTTPUpdateUser())
		r.Delete("/", api.HTTPDeleteUser())
	})

	return r
}

func newInfoRouter(api API) chi.Router {
	optAuth := OptionalAuth(api.Backend.DB.Users(), api.Secret, api.UnauthDelay, dao.User{})

	r := chi.NewRouter()

	r.With(optAuth).Get("/", Endpoint(api.epGetInfo))

	return r
}

// RedirectNoTrailingSlash is an http.HandlerFunc that redirects to the same URL as the
// request but with no trailing slash.
func RedirectNoTrailingSlash(w http.ResponseWriter, req *http.Request) {
	redirPath := strings.TrimRight(req.URL.Path, "/")
	redirection(redirPath).writeResponse(w, req)
}

func panicTo500(w http.ResponseWriter, req *http.Request) (panicRecovered bool) {
	if panicErr := recover(); panicErr != nil {
		textErr(
			http.StatusInternalServerError,
			"An internal server error occurred",
			fmt.Sprintf("panic: %v\nSTACK TRACE: %s", panicErr, string(debug.Stack())),
		).writeResponse(w, req)
		return true
	}
	return false
}
