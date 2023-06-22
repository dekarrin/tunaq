package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	APIPathPrefix = "/api/v1"
)

func (tqs *TunaQuestServer) initHandlers() {
	tqs.srv.HandleFunc("/", tqs.handlePathRoot)
	tqs.srv.HandleFunc(APIPathPrefix+"/login", tqs.handlePathLogin)
	tqs.srv.HandleFunc(APIPathPrefix+"/api/v1/login/", tqs.handlePathLogin)
	tqs.srv.HandleFunc(APIPathPrefix+"/api/v1/tokens", tqs.handlePathToken)
	tqs.srv.HandleFunc(APIPathPrefix+"/api/v1/tokens/", tqs.handlePathToken)
	tqs.srv.HandleFunc(APIPathPrefix+"/api/v1/users", tqs.handlePathUsers)
	tqs.srv.HandleFunc(APIPathPrefix+"/api/v1/users/", tqs.handlePathUsers)
}

func (tqs TunaQuestServer) handlePathRoot(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result endpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	result = jsonNotFound()
}

func (tqs TunaQuestServer) handlePathLogin(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result endpointResult
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
	var result endpointResult
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
	var result endpointResult
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
