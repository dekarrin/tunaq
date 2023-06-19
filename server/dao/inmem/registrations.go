package inmem

import (
	"context"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewRegistrationsRepository() *InMemoryRegistrationsRepository {
	return &InMemoryRegistrationsRepository{
		regs:          make(map[uuid.UUID]dao.Registration),
		byUserIDIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

type InMemoryRegistrationsRepository struct {
	regs          map[uuid.UUID]dao.Registration
	byUserIDIndex map[uuid.UUID][]uuid.UUID
}

func (imur *InMemoryRegistrationsRepository) Close() error {
	return nil
}

func (imur *InMemoryRegistrationsRepository) Create(ctx context.Context, reg dao.Registration) (dao.Registration, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Registration{}, fmt.Errorf("could not generate ID: %w", err)
	}

	reg.ID = newUUID
	reg.Created = time.Now()

	imur.regs[reg.ID] = reg

	userRegs := imur.byUserIDIndex[reg.UserID]
	if !util.InSlice(reg.ID, userRegs) {
		userRegs = append(userRegs, reg.ID)
		imur.byUserIDIndex[reg.UserID] = userRegs
	}

	return reg, nil
}

func (imur *InMemoryRegistrationsRepository) GetAll(ctx context.Context) ([]dao.Registration, error) {
	all := make([]dao.Registration, len(imur.regs))

	i := 0
	for k := range imur.regs {
		all[i] = imur.regs[k]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Registration) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemoryRegistrationsRepository) Update(ctx context.Context, id uuid.UUID, reg dao.Registration) (dao.Registration, error) {
	existing, ok := imur.regs[id]
	if !ok {
		return dao.Registration{}, dao.ErrNotFound
	}

	// check for conflicts
	if reg.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.regs[reg.ID]; ok {
			return dao.Registration{}, dao.ErrConstraintViolation
		}
	}

	imur.regs[reg.ID] = reg
	if reg.ID != id {
		delete(imur.regs, id)
		// also update it in the index slice
		byUser := imur.byUserIDIndex[existing.UserID]
		byUser = util.SliceIndexOf[]()
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
