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
)

func (tqs *TunaQuestServer) initHandlers() {
	tqs.srv.HandleFunc("/", tqs.handlePathRoot)
	tqs.srv.HandleFunc("/"+EntityLogin, tqs.handlePathLogin)
	tqs.srv.HandleFunc("/"+EntityLogin+"/", tqs.handlePathLogin)
}

func (tqs TunaQuestServer) handlePathRoot(w http.ResponseWriter, req *http.Request) {
	// this must be at the top of every handlePath* method to convert panics to
	// HTTP-500
	defer panicTo500(w, req)
	var result endpointResult
	defer func() {
		result.writeResponse(w, req)
	}()

	result = jsonErr(http.StatusNotFound, "The requested resource was not found", "not found")
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
		if req.Method == http.MethodPost {
			result = tqs.doEndpointLoginPOST(req)
		} else {
			result = jsonErr(http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
		}
	} else {
		// check for /login/{id}
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) != 2 {
			result = jsonErr(http.StatusNotFound, "The requested resource was not found", "not found")
			return
		}

		id, err := uuid.Parse(pathParts[1])
		if err != nil {
			result = jsonErr(http.StatusNotFound, "The requested resource was not found", "not found")
			return
		}

		if req.Method == http.MethodDelete {
			result = tqs.doEndpointLoginDELETE(req, id)
		} else {
			result = jsonErr(http.StatusMethodNotAllowed, "Method "+req.Method+" is not valid for "+req.URL.Path, "method not allowed")
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