package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

	tok, err := getJWT(req)
	if err != nil {
		// deliberately leaving as embedded if instead of &&
		if ah.required {
			// error here means token isn't present (or at least isn't in the
			// expected format, which for all intents and purposes is non-existent).
			// This is not okay if auth is required.

			result := jsonUnauthorized("", err.Error())
			time.Sleep(ah.unauthedDelay)
			result.writeResponse(w, req)
			return
		}
	} else {
		// validate the token
		lookupUser, err := validateAndLookupJWTUser(req.Context(), tok, ah.secret, ah.db)
		if err != nil {
			// deliberately leaving as embedded if instead of &&
			if ah.required {
				// there was a validation error. the user does not count as logged in.
				// if logging in is required, that's not okay.

				result := jsonUnauthorized("", err.Error())
				time.Sleep(ah.unauthedDelay)
				result.writeResponse(w, req)
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

func RequireAuth(db dao.UserRepository, secret []byte, unauthDelay time.Duration, defaultUser dao.User, next http.Handler) *AuthHandler {
	return &AuthHandler{
		db:            db,
		secret:        secret,
		unauthedDelay: unauthDelay,
		defaultUser:   dao.User{},
		required:      true,
		next:          next,
	}
}

func OptionalAuth(db dao.UserRepository, secret []byte, unauthDelay time.Duration, defaultUser dao.User, next http.Handler) *AuthHandler {
	return &AuthHandler{
		db:            db,
		secret:        secret,
		unauthedDelay: unauthDelay,
		defaultUser:   dao.User{},
		required:      false,
		next:          next,
	}
}

func validateAndLookupJWTUser(ctx context.Context, tok string, secret []byte, db dao.UserRepository) (dao.User, error) {
	var user dao.User

	_, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		// who is the user? we need this for further verification
		subj, err := t.Claims.GetSubject()
		if err != nil {
			return nil, fmt.Errorf("cannot get subject: %w", err)
		}

		id, err := uuid.Parse(subj)
		if err != nil {
			return nil, fmt.Errorf("cannot parse subject UUID: %w", err)
		}

		user, err = db.GetByID(ctx, id)
		if err != nil {
			if err == dao.ErrNotFound {
				return nil, fmt.Errorf("subject does not exist")
			} else {
				return nil, fmt.Errorf("subject could not be validated")
			}
		}

		var signKey []byte
		signKey = append(signKey, secret...)
		signKey = append(signKey, []byte(user.Password)...)
		signKey = append(signKey, []byte(fmt.Sprintf("%d", user.LastLogoutTime.Unix()))...)
		return signKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}), jwt.WithIssuer("tqs"), jwt.WithLeeway(time.Minute))

	if err != nil {
		return dao.User{}, err
	}

	return user, nil
}

func (tqs TunaQuestServer) requireJWT(ctx context.Context, req *http.Request) (dao.User, error) {
	tok, err := getJWT(req)
	if err != nil {
		return dao.User{}, err
	}

	return validateAndLookupJWTUser(ctx, tok, tqs.jwtSecret, tqs.db.Users())
}

func getJWT(req *http.Request) (string, error) {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))

	if authHeader == "" {
		return "", fmt.Errorf("no authorization header present")
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	scheme := strings.TrimSpace(strings.ToLower(authParts[0]))
	token := strings.TrimSpace(authParts[1])

	if scheme != "bearer" {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	return token, nil
}

func (tqs TunaQuestServer) generateJWT(u dao.User) (string, error) {
	claims := &jwt.MapClaims{
		"iss":        "tqs",
		"exp":        time.Now().Add(time.Hour).Unix(),
		"sub":        u.ID.String(),
		"authorized": true,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	var signKey []byte
	signKey = append(signKey, tqs.jwtSecret...)
	signKey = append(signKey, []byte(u.Password)...)
	signKey = append(signKey, []byte(fmt.Sprintf("%d", u.LastLogoutTime.Unix()))...)

	tokStr, err := tok.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return tokStr, nil
}
