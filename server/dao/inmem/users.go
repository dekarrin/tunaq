package inmem

import (
	"context"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewUsersRepository() *InMemoryUsersRepository {
	return &InMemoryUsersRepository{
		users:           make(map[uuid.UUID]dao.User),
		byUsernameIndex: make(map[string]uuid.UUID),
	}
}

type InMemoryUsersRepository struct {
	users           map[uuid.UUID]dao.User
	byUsernameIndex map[string]uuid.UUID
}

func (imur *InMemoryUsersRepository) Create(ctx context.Context, user dao.User) (dao.User, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.User{}, fmt.Errorf("could not generate ID: %w", err)
	}

	user.ID = newUUID

	// make sure it's not already in the DB
	if _, ok := imur.byUsernameIndex[user.Username]; ok {
		return dao.User{}, dao.ErrConstraintViolation
	}

	user.LastLogoutTime = time.Now()

	imur.users[user.ID] = user
	imur.byUsernameIndex[user.Username] = user.ID

	return user, nil
}

func (imur *InMemoryUsersRepository) Update(ctx context.Context, id uuid.UUID, user dao.User) (dao.User, error) {
	existing, ok := imur.users[id]
	if !ok {
		return dao.User{}, dao.ErrNotFound
	}

	if user.Username != existing.Username {
		// that's okay but we need to check it
		if _, ok := imur.byUsernameIndex[user.Username]; ok {
			return dao.User{}, dao.ErrConstraintViolation
		}
	} else if user.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.users[id]; ok {
			return dao.User{}, dao.ErrConstraintViolation
		}
	}

	imur.users[user.ID] = user
	imur.byUsernameIndex[user.Username] = user.ID
	if user.ID != id {
		delete(imur.users, id)
	}

	return user, nil
}

func (imur *InMemoryUsersRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.User, error) {
	user, ok := imur.users[id]
	if !ok {
		return dao.User{}, dao.ErrNotFound
	}

	return user, nil
}

func (imur *InMemoryUsersRepository) GetByUsername(ctx context.Context, username string) (dao.User, error) {
	userID, ok := imur.byUsernameIndex[username]
	if !ok {
		return dao.User{}, dao.ErrNotFound
	}

	return imur.users[userID], nil
}

func (imur *InMemoryUsersRepository) Delete(ctx context.Context, id uuid.UUID) (dao.User, error) {
	user, ok := imur.users[id]
	if !ok {
		return dao.User{}, dao.ErrNotFound
	}

	delete(imur.byUsernameIndex, user.Username)
	delete(imur.users, user.ID)

	return user, nil
}
