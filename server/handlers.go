package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
)

const (
	EntityLogin = "login"
	EntityToken = "tokens"
)

func (tqs *TunaQuestServer) initHandlers() {
	tqs.srv.HandleFunc("/", tqs.handlePathRoot)
	tqs.srv.HandleFunc("/"+EntityLogin, tqs.handlePathLogin)
	tqs.srv.HandleFunc("/"+EntityLogin+"/", tqs.handlePathLogin)
	tqs.srv.HandleFunc("/"+EntityToken, tqs.handlePathToken)
	tqs.srv.HandleFunc("/"+EntityToken+"/", tqs.handlePathToken)
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

	if req.URL.Path == "/"+EntityLogin+"/" || req.URL.Path == "/"+EntityLogin {

		// ---------------------------------------------- //
		// DISPATCH FOR: /login                           //
		// ---------------------------------------------- //
		if req.Method == http.MethodPost {
			result = tqs.doEndpoint_Login_POST(req)
		} else {
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
		if req.Method == http.MethodDelete {
			result = tqs.doEndpoint_LoginID_DELETE(req, id)
		} else {
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

	if req.URL.Path == "/"+EntityToken+"/" || req.URL.Path == "/"+EntityToken {

		// ---------------------------------------------- //
		// DISPATCH FOR: /tokens                          //
		// ---------------------------------------------- //
		if req.Method == http.MethodPost {
			result = tqs.doEndpoint_Token_POST(req)
		} else {
			result = jsonMethodNotAllowed(req)
		}
	} else {
		result = jsonNotFound()
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
