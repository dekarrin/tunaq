package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	//users := newUsersRouter(service)
	info := newInfoRouter(service)

	r.Mount("/login", login)
	r.Mount("/tokens", tokens)
	// r.Mount("/users", users)
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
	r := chi.NewRouter()

	r.Post("/", Endpoint(service.doEndpoint_Login_POST).ServeHTTP)
	r.Delete("/"+p("id:uuid"), Endpoint(service.deleteLogin).ServeHTTP)
	r.HandleFunc("/"+p("id:uuid")+"/", RedirectNoTrailingSlash)

	return r
}

func newTokensRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	r.Post("/", Endpoint(service.doEndpoint_Tokens_POST).ServeHTTP)

	return r
}

func newUsersRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	return r
}

func newInfoRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	r.Get("/", Endpoint(service.doEndpoint_Info_GET).ServeHTTP)

	return r
}

func (tqs *TunaQuestServer) initHandlers() {
	tqs.srv.HandleFunc("/", tqs.handlePathRoot)
	tqs.srv.HandleFunc(APIPathPrefix+"/login", tqs.handlePathLogin)
	tqs.srv.HandleFunc(APIPathPrefix+"/login/", tqs.handlePathLogin)
	tqs.srv.HandleFunc(APIPathPrefix+"/tokens", tqs.handlePathToken)
	tqs.srv.HandleFunc(APIPathPrefix+"/tokens/", tqs.handlePathToken)
	tqs.srv.HandleFunc(APIPathPrefix+"/users", tqs.handlePathUsers)
	tqs.srv.HandleFunc(APIPathPrefix+"/users/", tqs.handlePathUsers)
	tqs.srv.HandleFunc(APIPathPrefix+"/info", tqs.handlePathInfo)
	tqs.srv.HandleFunc(APIPathPrefix+"/info/", tqs.handlePathInfo)
}

// RedirectNoTrailingSlash is an http.HandlerFunc that redirects to the same URL as the
// request but with no trailing slash.
func RedirectNoTrailingSlash(w http.ResponseWriter, req *http.Request) {
	redirPath := strings.TrimRight(req.URL.Path, "/")
	redirection(redirPath).writeResponse(w, req)
}

func (tqs TunaQuestServer) handlePathInfo(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	if req.URL.Path == APIPathPrefix+"/info/" || req.URL.Path == APIPathPrefix+"/info" {

		// ---------------------------------------------- //
		// DISPATCH FOR: /info                            //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodGet:
			result = tqs.doEndpoint_Info_GET(req)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	}
}

func (tqs TunaQuestServer) handlePathRoot(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	result = jsonNotFound()
}

func (tqs TunaQuestServer) handlePathLogin(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	if req.URL.Path == APIPathPrefix+"/login/" || req.URL.Path == APIPathPrefix+"/login" {

		// ---------------------------------------------- //
		// DISPATCH FOR: /login                           //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodPost:
			result = tqs.doEndpoint_Login_POST(req)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	} else {
		// check for /login/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			result = jsonNotFound()
			return
		}

		id, err := uuid.Parse(pathParts[1])
		if err != nil {
			result = jsonNotFound()
			return
		}

		// ---------------------------------------------- //
		// DISPATCH FOR: /login/{id}                      //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodDelete:
			result = tqs.doEndpoint_LoginID_DELETE(req, id)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	}
}

func (tqs TunaQuestServer) handlePathToken(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	if req.URL.Path == APIPathPrefix+"/tokens/" || req.URL.Path == APIPathPrefix+"/tokens" {

		// ---------------------------------------------- //
		// DISPATCH FOR: /tokens                          //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodPost:
			result = tqs.doEndpoint_Tokens_POST(req)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	} else {
		result = jsonNotFound()
	}
}

func (tqs TunaQuestServer) handlePathUsers(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result EndpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	if req.URL.Path == APIPathPrefix+"/users/" || req.URL.Path == APIPathPrefix+"/users" {

		// ---------------------------------------------- //
		// DISPATCH FOR: /users                           //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodPost:
			result = tqs.doEndpoint_Users_POST(req)
		case http.MethodGet:
			result = tqs.doEndpoint_Users_GET(req)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	} else {
		// check for /users/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			result = jsonNotFound()
			return
		}

		id, err := uuid.Parse(pathParts[1])
		if err != nil {
			result = jsonNotFound()
			return
		}

		// ---------------------------------------------- //
		// DISPATCH FOR: /users/{id}                      //
		// ---------------------------------------------- //
		switch req.Method {
		case http.MethodGet:
			result = tqs.doEndpoint_UsersID_GET(req, id)
		case http.MethodPut:
			result = tqs.doEndpoint_UsersID_PUT(req, id)
		case http.MethodPatch:
			result = tqs.doEndpoint_UsersID_PATCH(req, id)
		case http.MethodDelete:
			result = tqs.doEndpoint_UsersID_DELETE(req, id)
		default:
			time.Sleep(tqs.unauthedDelay)
			result = jsonMethodNotAllowed(req)
		}
	}
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
