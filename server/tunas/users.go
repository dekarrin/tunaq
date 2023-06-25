package tunas

import (
	"context"
	"encoding/base64"
	"errors"
	"net/mail"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/dekarrin/tunaq/server/serr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// GetAllUsers returns all users currently in persistence.
func (svc Service) GetAllUsers(ctx context.Context) ([]dao.User, error) {
	users, err := svc.DB.Users().GetAll(ctx)
	if err != nil {
		return nil, serr.WrapDB("", err)
	}

	return users, nil
}

// GetUser returns the user with the given ID.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with that ID exists,
// it will match serr.ErrNotFound. If the error occured due to an unexpected
// problem with the DB, it will match serr.ErrDB. Finally, if there is an issue
// with one of the arguments, it will match serr.ErrBadArgument.
func (svc Service) GetUser(ctx context.Context, id string) (dao.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	user, err := svc.DB.Users().GetByID(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not get user", err)
	}

	return user, nil
}

// CreateUser creates a new user with the given username, password, and email
// combo. Returns the newly-created user as it exists after creation.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If a user with that username is
// already present, it will match serr.ErrAlreadyExists. If the error occured
// due to an unexpected problem with the DB, it will match serr.ErrDB. Finally,
// if one of the arguments is invalid, it will match serr.ErrBadArgument.
func (svc Service) CreateUser(ctx context.Context, username, password, email string, role dao.Role) (dao.User, error) {
	var err error
	if username == "" {
		return dao.User{}, serr.New("username cannot be blank", err, serr.ErrBadArgument)
	}
	if password == "" {
		return dao.User{}, serr.New("password cannot be blank", err, serr.ErrBadArgument)
	}

	var storedEmail *mail.Address
	if email != "" {
		storedEmail, err = mail.ParseAddress(email)
		if err != nil {
			return dao.User{}, serr.New("email is not valid", err, serr.ErrBadArgument)
		}
	}

	_, err = svc.DB.Users().GetByUsername(ctx, username)
	if err == nil {
		return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
	} else if !errors.Is(err, dao.ErrNotFound) {
		return dao.User{}, serr.WrapDB("", err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, serr.New("password is too long", err, serr.ErrBadArgument)
		} else {
			return dao.User{}, serr.New("password could not be encrypted", err)
		}
	}

	storedPass := base64.StdEncoding.EncodeToString(passHash)

	newUser := dao.User{
		Username: username,
		Password: storedPass,
		Email:    storedEmail,
		Role:     role,
	}

	user, err := svc.DB.Users().Create(ctx, newUser)
	if err != nil {
		if errors.Is(err, dao.ErrConstraintViolation) {
			return dao.User{}, serr.ErrAlreadyExists
		}
		return dao.User{}, serr.WrapDB("could not create user", err)
	}

	return user, nil
}

// UpdateUser sets the properties of the user with the given ID to the
// properties in the given user. All the given properties of the user will
// overwrite the existing ones. Returns the updated user.
//
// This function cannot be used to update the password. Use UpdatePassword for
// that.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If a user with that username or
// ID (if they are changing) is already present, it will match
// serr.ErrAlreadyExists. If no user with the given ID exists, it will match
// serr.ErrNotFound. If the error occured due to an unexpected problem with the
// DB, it will match serr.ErrDB. Finally, if one of the arguments is invalid, it
// will match serr.ErrBadArgument.
func (svc Service) UpdateUser(ctx context.Context, curID, newID, username, email string, role dao.Role) (dao.User, error) {
	var err error

	if username == "" {
		return dao.User{}, serr.New("username cannot be blank", err, serr.ErrBadArgument)
	}

	var storedEmail *mail.Address
	if email != "" {
		storedEmail, err = mail.ParseAddress(email)
		if err != nil {
			return dao.User{}, serr.New("email is not valid", err, serr.ErrBadArgument)
		}
	}

	uuidCurID, err := uuid.Parse(curID)
	if err != nil {
		return dao.User{}, serr.New("current ID is not valid", serr.ErrBadArgument)
	}
	uuidNewID, err := uuid.Parse(newID)
	if err != nil {
		return dao.User{}, serr.New("new ID is not valid", serr.ErrBadArgument)
	}

	daoUser, err := svc.DB.Users().GetByID(ctx, uuidCurID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("user not found", serr.ErrNotFound)
		}
	}

	if curID != newID {
		_, err := svc.DB.Users().GetByID(ctx, uuidNewID)
		if err == nil {
			return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
		} else if !errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.WrapDB("", err)
		}
	}
	if daoUser.Username != username {
		_, err := svc.DB.Users().GetByUsername(ctx, username)
		if err == nil {
			return dao.User{}, serr.New("a user with that username already exists", serr.ErrAlreadyExists)
		} else if !errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.WrapDB("", err)
		}
	}

	daoUser.Email = storedEmail
	daoUser.ID = uuidNewID
	daoUser.Username = username
	daoUser.Role = role

	updatedUser, err := svc.DB.Users().Update(ctx, uuidCurID, daoUser)
	if err != nil {
		if errors.Is(err, dao.ErrConstraintViolation) {
			return dao.User{}, serr.New("a user with that ID/username already exists", serr.ErrAlreadyExists)
		} else if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("user not found", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	return updatedUser, nil
}

// UpdatePassword sets the password of the user with the given ID to the new
// password. The new password cannot be empty. Returns the updated user.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with the given ID
// exists, it will match serr.ErrNotFound. If the error occured due to an
// unexpected problem with the DB, it will match serr.ErrDB. Finally, if one of
// the arguments is invalid, it will match serr.ErrBadArgument.
func (svc Service) UpdatePassword(ctx context.Context, id, password string) (dao.User, error) {
	if password == "" {
		return dao.User{}, serr.New("password cannot be empty", serr.ErrBadArgument)
	}
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	existing, err := svc.DB.Users().GetByID(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("no user with that ID exists", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("", err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		if err == bcrypt.ErrPasswordTooLong {
			return dao.User{}, serr.New("password is too long", err, serr.ErrBadArgument)
		} else {
			return dao.User{}, serr.New("password could not be encrypted", err)
		}
	}

	storedPass := base64.StdEncoding.EncodeToString(passHash)

	existing.Password = storedPass

	updated, err := svc.DB.Users().Update(ctx, uuidID, existing)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.New("no user with that ID exists", serr.ErrNotFound)
		}
		return dao.User{}, serr.WrapDB("could not update user", err)
	}

	return updated, nil
}

// DeleteUser deletes the user with the given ID. It returns the deleted user
// just after they were deleted.
//
// The returned error, if non-nil, will return true for various calls to
// errors.Is depending on what caused the error. If no user with that username
// exists, it will match serr.ErrNotFound. If the error occured due to an
// unexpected problem with the DB, it will match serr.ErrDB. Finally, if there
// is an issue with one of the arguments, it will match serr.ErrBadArgument.
func (svc Service) DeleteUser(ctx context.Context, id string) (dao.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return dao.User{}, serr.New("ID is not valid", serr.ErrBadArgument)
	}

	user, err := svc.DB.Users().Delete(ctx, uuidID)
	if err != nil {
		if errors.Is(err, dao.ErrNotFound) {
			return dao.User{}, serr.ErrNotFound
		}
		return dao.User{}, serr.WrapDB("could not delete user", err)
	}

	return user, nil
}
