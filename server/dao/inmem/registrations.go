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
	userRegs = append(userRegs, reg.ID)
	imur.byUserIDIndex[reg.UserID] = userRegs

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

func (imur *InMemoryRegistrationsRepository) GetAllByUserID(ctx context.Context, id uuid.UUID) ([]dao.Registration, error) {
	byUser := imur.byUserIDIndex[id]
	if len(byUser) < 1 {
		return nil, dao.ErrNotFound
	}

	all := make([]dao.Registration, len(byUser))

	i := 0
	for k := range imur.regs {
		if imur.regs[k]
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

		// also update it in the index slice if we are not about to remove it
		if existing.UserID == reg.UserID {
			byUser := imur.byUserIDIndex[existing.UserID]
			pos := util.SliceIndexOf(id, byUser)
			if pos < 0 {
				return dao.Registration{}, fmt.Errorf("DB ASSERTION FAILURE: missing index entry for user %s to reg %s", existing.UserID, existing.ID)
			}
			byUser[pos] = reg.ID
			imur.byUserIDIndex[existing.UserID] = byUser
		}
	}

	if reg.UserID != existing.UserID {
		// if we're modifying the user, we must remove it from old index
		// entry and put it into another.
		byUser := imur.byUserIDIndex[existing.UserID]
		updated := util.SliceRemove(existing.ID, byUser)
		imur.byUserIDIndex[existing.UserID] = updated
		if len(updated) < 1 {
			delete(imur.byUserIDIndex, existing.UserID)
		}

		newByUser := imur.byUserIDIndex[reg.UserID]
		newByUser = append(newByUser, reg.ID)
		imur.byUserIDIndex[reg.UserID] = newByUser
	}

	return reg, nil
}

func (imur *InMemoryRegistrationsRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.Registration, error) {
	reg, ok := imur.regs[id]
	if !ok {
		return dao.Registration{}, dao.ErrNotFound
	}

	return reg, nil
}

func (imur *InMemoryRegistrationsRepository) Delete(ctx context.Context, id uuid.UUID) (dao.Registration, error) {
	reg, ok := imur.regs[id]
	if !ok {
		return dao.Registration{}, dao.ErrNotFound
	}

	byUser := imur.byUserIDIndex[reg.UserID]
	updated := util.SliceRemove(reg.ID, byUser)
	imur.byUserIDIndex[reg.UserID] = updated
	if len(updated) < 1 {
		delete(imur.byUserIDIndex, reg.UserID)
	}
	delete(imur.regs, reg.ID)

	return reg, nil
}
