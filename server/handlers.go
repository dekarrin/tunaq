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

const (
	URLParamKeyID = "id"
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

func newRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	r.Mount(APIPathPrefix, newAPIRouter(service))

	return r
}

func newAPIRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	login := newLoginRouter(service)
	tokens := newTokensRouter(service)
	users := newUsersRouter(service)
	info := newInfoRouter(service)

	r.Mount("/login", login)
	r.Mount("/tokens", tokens)
	r.Mount("/users", users)
	r.Mount("/info", info)
	r.HandleFunc("/info/", RedirectNoTrailingSlash)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		jsonNotFound().writeResponse(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(service.unauthedDelay)
		jsonMethodNotAllowed(r).writeResponse(w, r)
	})

	return r
}

func newLoginRouter(service *TunaQuestServer) chi.Router {
	reqAuth := RequireAuth(service.db.Users(), service.jwtSecret, service.unauthedDelay, dao.User{})

	r := chi.NewRouter()

	r.Post("/", Endpoint(service.epCreateLogin))
	r.With(reqAuth).Delete("/"+p("id:uuid"), Endpoint(service.epDeleteLogin))
	r.HandleFunc("/"+p("id:uuid")+"/", RedirectNoTrailingSlash)

	return r
}

func newTokensRouter(service *TunaQuestServer) chi.Router {
	reqAuth := RequireAuth(service.db.Users(), service.jwtSecret, service.unauthedDelay, dao.User{})

	r := chi.NewRouter()

	r.With(reqAuth).Post("/", Endpoint(service.epCreateToken))

	return r
}

func newUsersRouter(service *TunaQuestServer) chi.Router {
	reqAuth := RequireAuth(service.db.Users(), service.jwtSecret, service.unauthedDelay, dao.User{})

	r := chi.NewRouter()

	r.Use(reqAuth)

	r.Get("/", Endpoint(service.epGetAllUsers))
	r.Post("/", Endpoint(service.epCreateNewUser))

	r.Route("/"+p("id:uuid"), func(r chi.Router) {
		r.Get("/", Endpoint(service.epGetUser))
		r.Put("/", Endpoint(service.epCreateExistingUser))
		r.Patch("/", Endpoint(service.epUpdateUser))
		r.Delete("/", Endpoint(service.epDeleteUser))
	})

	return r
}

func newInfoRouter(service *TunaQuestServer) chi.Router {
	optAuth := OptionalAuth(service.db.Users(), service.jwtSecret, service.unauthedDelay, dao.User{})

	r := chi.NewRouter()

	r.With(optAuth).Get("/", Endpoint(service.epGetInfo))

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
