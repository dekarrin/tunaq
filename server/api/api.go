// Package API provides HTTP API endpoints for the TunaQuest server.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/result"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/dekarrin/tunaq/server/tunas"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	// PathPrefix is the prefix of all paths in the API. Routers should mount
	// a sub-router that routes all requests to the API at this path.
	PathPrefix = "/api/v1"
)

// requireIDParam gets the ID of the main entity being referenced in the URI and
// returns it. It panics if the key is not there or is not parsable.
func requireIDParam(r *http.Request) uuid.UUID {
	id, err := getURLParam(r, "id", uuid.Parse)
	if err != nil {
		panic(err.Error())
	}
	return id
}

func getURLParam[E any](r *http.Request, key string, parse func(string) (E, error)) (val E, err error) {
	valStr := chi.URLParam(r, key)
	if valStr == "" {
		// either it does not exist or it is nil; treat both as the same and
		// return an error
		return val, fmt.Errorf("parameter does not exist")
	}

	val, err = parse(valStr)
	if err != nil {
		return val, serr.New("", serr.ErrBadArgument)
	}
	return val, nil
}

// API holds parameters for endpoints needed to run and a service layer that
// will perform most of the actual logic. To use API, create one and then
// assign the result of its HTTP* methods as handlers to a router or some other
// kind of server mux.
//
// This is exclusively an API for serving external requests. For direct
// programmatic access into the backend of a TunaQuest server via Go code, see
// [tunas.Service].
type API struct {
	// Backend is the service that the API calls to perform the requested
	// actions.
	Backend tunas.Service

	// UnauthDelay is the amount of time that a request will pause before
	// responding with an HTTP-403, HTTP-401, or HTTP-500 to deprioritize such
	// requests from processing and I/O.
	UnauthDelay time.Duration

	// Secret is the secret used to sign JWT tokens.
	Secret []byte
}

// v must be a pointer to a type. Will return error such that
// errors.Is(err, ErrMalformedBody) returns true if it is problem decoding the
// JSON itself.
func parseJSON(req *http.Request, v interface{}) error {
	contentType := req.Header.Get("Content-Type")

	if strings.ToLower(contentType) != "application/json" {
		return fmt.Errorf("request content-type is not application/json")
	}

	bodyData, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}
	defer func() {
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyData))
	}()

	err = json.Unmarshal(bodyData, v)
	if err != nil {
		return serr.New("malformed JSON in request", err, serr.ErrBodyUnmarshal)
	}

	return nil
}

type EndpointFunc func(req *http.Request) result.Result

func httpEndpoint(unauthDelay time.Duration, ep EndpointFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		defer panicTo500(w, req)
		result := ep(req)

		if result.Status == http.StatusUnauthorized || result.Status == http.StatusForbidden || result.Status == http.StatusInternalServerError {
			// if it's one of these statusus, either the user is improperly
			// logging in or tried to access a forbidden resource, both of which
			// should force the wait time before responding.
			time.Sleep(unauthDelay)
		}

		// TODO: pull logging out of writeResponse and do that immediately after
		// getting the result so it isn't subject to the wait time.
		result.WriteResponse(w, req)
	}
}

func panicTo500(w http.ResponseWriter, req *http.Request) (panicRecovered bool) {
	if panicErr := recover(); panicErr != nil {
		result.TextErr(
			http.StatusInternalServerError,
			"An internal server error occurred",
			fmt.Sprintf("panic: %v\nSTACK TRACE: %s", panicErr, string(debug.Stack())),
		).WriteResponse(w, req)
		return true
	}
	return false
}
