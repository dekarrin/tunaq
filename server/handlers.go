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

func newRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	r.Mount(APIPathPrefix, newAPIRouter(service))

	return r
}

func newAPIRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	//login := newLoginRouter()
	//tokens := newTokensRouter()
	//users := newUsersRouter()
	info := newInfoRouter(service)

	// r.Mount("/login", login)
	// r.Mount("/tokens", tokens)
	// r.Mount("/users", users)
	r.Mount("/info", info)
	r.HandleFunc("/info/", RedirectNoTrailingSlash)

	return r
}

func newLoginRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

	return r
}

func newTokensRouter(service *TunaQuestServer) chi.Router {
	r := chi.NewRouter()

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

func panicTo500(w http.ResponseWriter, req *http.Request) {
	if panicErr := recover(); panicErr != nil {
		textErr(
			http.StatusInternalServerError,
			"An internal server error occurred",
			fmt.Sprintf("panic: %v\n%s", panicErr, string(debug.Stack())),
		).writeResponse(w, req)
	}
}
