// Package middle contains middleware for use with the TunaQuest server.
package middle

import (
	"context"
	"net/http"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/result"
	"github.com/dekarrin/tunaq/server/token"
)

// Middleware is a function that takes a handler and returns a new handler which
// wraps the given one and provides some additional functionality.
type Middleware func(next http.Handler) http.Handler

// AuthKey is a key in the context of a request populated by an AuthHandler.
type AuthKey int64

const (
	AuthLoggedIn AuthKey = iota
	AuthUser
)

// AuthHandler is middleware that will accept a request, extract the token used
// for authentication, and make calls to get a User entity that represents the
// logged in user from the token.
//
// Keys are added to the request context before the request is passed to the
// next step in the chain. AuthUser will contain the logged-in user, and
// AuthLoggedIn will return whether the user is logged in (only applies for
// optional logins; for non-optional, not being logged in will result in an
// HTTP error being returned before the request is passed to the next handler).
type AuthHandler struct {
	db            dao.UserRepository
	secret        []byte
	required      bool
	defaultUser   dao.User
	unauthedDelay time.Duration
	next          http.Handler
}

func (ah *AuthHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var loggedIn bool
	user := ah.defaultUser

	tok, err := token.Get(req)
	if err != nil {
		// deliberately leaving as embedded if instead of &&
		if ah.required {
			// error here means token isn't present (or at least isn't in the
			// expected format, which for all intents and purposes is non-existent).
			// This is not okay if auth is required.

			result := result.Unauthorized("", err.Error())
			time.Sleep(ah.unauthedDelay)
			result.WriteResponse(w, req)
			return
		}
	} else {
		// validate the token
		lookupUser, err := token.Validate(req.Context(), tok, ah.secret, ah.db)
		if err != nil {
			// deliberately leaving as embedded if instead of &&
			if ah.required {
				// there was a validation error. the user does not count as logged in.
				// if logging in is required, that's not okay.

				result := result.Unauthorized("", err.Error())
				time.Sleep(ah.unauthedDelay)
				result.WriteResponse(w, req)
				return
			}
		} else {
			user = lookupUser
			loggedIn = true
		}
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, AuthLoggedIn, loggedIn)
	ctx = context.WithValue(ctx, AuthUser, user)
	req = req.WithContext(ctx)
	ah.next.ServeHTTP(w, req)
}

func RequireAuth(db dao.UserRepository, secret []byte, unauthDelay time.Duration, defaultUser dao.User) Middleware {
	return func(next http.Handler) http.Handler {
		return &AuthHandler{
			db:            db,
			secret:        secret,
			unauthedDelay: unauthDelay,
			defaultUser:   dao.User{},
			required:      true,
			next:          next,
		}
	}
}

func OptionalAuth(db dao.UserRepository, secret []byte, unauthDelay time.Duration, defaultUser dao.User) Middleware {
	return func(next http.Handler) http.Handler {
		return &AuthHandler{
			db:            db,
			secret:        secret,
			unauthedDelay: unauthDelay,
			defaultUser:   dao.User{},
			required:      false,
			next:          next,
		}
	}
}
