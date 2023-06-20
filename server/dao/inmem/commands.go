package inmem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

// NewCommandsRepository creates a new Commands repo. If seshRepo is not
// provided, GetAllByUser() will always return nil.
func NewCommandsRepository(seshRepo dao.SessionRepository) *InMemoryCommandsRepository {
	return &InMemoryCommandsRepository{
		seshRepo:      seshRepo,
		coms:          make(map[uuid.UUID]dao.Command),
		bySeshIDIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

type InMemoryCommandsRepository struct {
	coms          map[uuid.UUID]dao.Command
	seshRepo      dao.SessionRepository
	bySeshIDIndex map[uuid.UUID][]uuid.UUID
}

func (imur *InMemoryCommandsRepository) Close() error {
	return nil
}

func (imur *InMemoryCommandsRepository) Create(ctx context.Context, c dao.Command) (dao.Command, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Command{}, fmt.Errorf("could not generate ID: %w", err)
	}

	c.ID = newUUID
	c.Created = time.Now()

	if imur.seshRepo != nil {
		_, err := imur.seshRepo.GetByID(ctx, c.SessionID)
		if err != nil {
			if errors.Is(err, dao.ErrNotFound) {
				return dao.Command{}, dao.ErrConstraintViolation
			} else {
				return dao.Command{}, err
			}
		}
	}

	imur.coms[c.ID] = c

	seshComs := imur.bySeshIDIndex[c.SessionID]
	seshComs = append(seshComs, c.ID)
	imur.bySeshIDIndex[c.SessionID] = seshComs

	return c, nil
}

func (imur *InMemoryCommandsRepository) GetAll(ctx context.Context) ([]dao.Command, error) {
	all := make([]dao.Command, len(imur.coms))

	i := 0
	for k := range imur.coms {
		all[i] = imur.coms[k]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Command) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemoryCommandsRepository) GetAllByUser(ctx context.Context, id uuid.UUID) ([]dao.Command, error) {
	if imur.seshRepo == nil {
		return nil, nil
	}

	userSessions, err := imur.seshRepo.GetAllByUser(ctx, id)
	if err != nil {
		return nil, err
	}

	allCommands := []dao.Command{}
	for _, sesh := range userSessions {
		seshCommands, err := imur.GetAllBySession(ctx, sesh.ID)
		if err != nil {
			return nil, err
		}
		allCommands = append(allCommands, seshCommands...)
	}

	return allCommands, nil
}

func (imur *InMemoryCommandsRepository) GetAllBySession(ctx context.Context, id uuid.UUID) ([]dao.Command, error) {
	bySesh := imur.bySeshIDIndex[id]
	if len(bySesh) < 1 {
		return nil, dao.ErrNotFound
	}

	all := make([]dao.Command, len(bySesh))

	for i := range bySesh {
		all[i] = imur.coms[bySesh[i]]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Command) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemoryCommandsRepository) Update(ctx context.Context, id uuid.UUID, c dao.Command) (dao.Command, error) {
	existing, ok := imur.coms[id]
	if !ok {
		return dao.Command{}, dao.ErrNotFound
	}

	// check for conflicts on this table only
	// (inmem does not support enforcement of foreign keys)
	if c.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.coms[c.ID]; ok {
			return dao.Command{}, dao.ErrConstraintViolation
		}
	}

	imur.coms[c.ID] = c
	if c.ID != id {
		delete(imur.coms, id)

		// also update it in the index slices if we are not about to remove it
		if existing.SessionID == c.SessionID {
			bySesh := imur.bySeshIDIndex[existing.SessionID]
			pos := util.SliceIndexOf(id, bySesh)
			if pos < 0 {
				return dao.Command{}, fmt.Errorf("DB ASSERTION FAILURE: missing index entry for sesh %s to sesh %s", existing.SessionID, existing.ID)
			}
			bySesh[pos] = c.ID
			imur.bySeshIDIndex[existing.SessionID] = bySesh
		}
	}

	if c.SessionID != existing.SessionID {
		// if we're modifying the user, we must remove it from old index
		// entry and put it into another.
		bySesh := imur.bySeshIDIndex[existing.SessionID]
		updated := util.SliceRemove(existing.ID, bySesh)
		imur.bySeshIDIndex[existing.SessionID] = updated
		if len(updated) < 1 {
			delete(imur.bySeshIDIndex, existing.SessionID)
		}

		newBySesh := imur.bySeshIDIndex[c.SessionID]
		newBySesh = append(newBySesh, c.ID)
		imur.bySeshIDIndex[c.SessionID] = newBySesh
	}

	return c, nil
}

func (imur *InMemoryCommandsRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.Command, error) {
	c, ok := imur.coms[id]
	if !ok {
		return dao.Command{}, dao.ErrNotFound
	}

	return c, nil
}

func (imur *InMemoryCommandsRepository) Delete(ctx context.Context, id uuid.UUID) (dao.Command, error) {
	c, ok := imur.coms[id]
	if !ok {
		return dao.Command{}, dao.ErrNotFound
	}

	bySesh := imur.bySeshIDIndex[c.SessionID]
	updated := util.SliceRemove(c.ID, bySesh)
	imur.bySeshIDIndex[c.SessionID] = updated
	if len(updated) < 1 {
		delete(imur.bySeshIDIndex, c.SessionID)
	}

	delete(imur.coms, c.ID)

	return c, nil
}
