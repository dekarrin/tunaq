package tunas

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Login verifies the provided username and password against the existing user
// in persistence and returns that user if they match. Returns the user entity
// from the persistence layer that the username and password are valid for.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the credentials do not match
// a user or if the password is incorrect, it will match ErrBadCredentials. If
// the error occured due to an unexpected problem with the DB, it will match
// serr.ErrDB.
func (svc Service) Login(ctx context.Context, username string, password string) (dao.User, error) {
	user, err := svc.DB.Users().GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrBadCredentials
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	// verify password
	bcryptHash, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		return dao.User{}, err
	}

	err = bcrypt.CompareHashAndPassword(bcryptHash, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return dao.User{}, serr.ErrBadCredentials
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	// successful login; update the DB
	user.LastLoginTime = time.Now()
	user, err = svc.DB.Users().Update(ctx, user.ID, user)
	if err != nil {
		return dao.User{}, serr.WrapDB("cannot update user login time", err)
	}

	return user, nil
}

// Logout marks the user with the given ID as having logged out, invalidating
// any login that may be active. Returns the user entity that was logged out.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If the user doesn't exist, it
// will match serr.ErrNotFound. If the error occured due to an unexpected
// problem with the DB, it will match serr.ErrDB.
func (svc Service) Logout(ctx context.Context, who uuid.UUID) (dao.User, error) {
	existing, err := svc.DB.Users().GetByID(ctx, who)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not retrieve user", err)
	}

	existing.LastLogoutTime = time.Now()

	updated, err := svc.DB.Users().Update(ctx, existing.ID, existing)
	if err != nil {
		return dao.User{}, serr.WrapDB("could not update user", err)
	}

	return updated, nil
}
