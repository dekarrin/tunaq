package inmem

import (
	"context"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewSessionsRepository() *InMemorySessionsRepository {
	return &InMemorySessionsRepository{
		seshes:        make(map[uuid.UUID]dao.Session),
		byUserIDIndex: make(map[uuid.UUID][]uuid.UUID),
		byGameIDIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

type InMemorySessionsRepository struct {
	seshes        map[uuid.UUID]dao.Session
	byUserIDIndex map[uuid.UUID][]uuid.UUID
	byGameIDIndex map[uuid.UUID][]uuid.UUID
}

func (imur *InMemorySessionsRepository) Close() error {
	return nil
}

func (imur *InMemorySessionsRepository) Create(ctx context.Context, s dao.Session) (dao.Session, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Session{}, fmt.Errorf("could not generate ID: %w", err)
	}

	s.ID = newUUID
	s.Created = time.Now()

	imur.seshes[s.ID] = s

	userSeshes := imur.byUserIDIndex[s.UserID]
	userSeshes = append(userSeshes, s.ID)
	imur.byUserIDIndex[s.UserID] = userSeshes

	gameSeshes := imur.byGameIDIndex[s.GameID]
	gameSeshes = append(gameSeshes, s.ID)
	imur.byGameIDIndex[s.GameID] = gameSeshes

	return s, nil
}

func (imur *InMemorySessionsRepository) GetAll(ctx context.Context) ([]dao.Session, error) {
	all := make([]dao.Session, len(imur.seshes))

	i := 0
	for k := range imur.seshes {
		all[i] = imur.seshes[k]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Session) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemorySessionsRepository) GetAllByUser(ctx context.Context, id uuid.UUID) ([]dao.Session, error) {
	byUser := imur.byUserIDIndex[id]
	if len(byUser) < 1 {
		return nil, dao.ErrNotFound
	}

	all := make([]dao.Session, len(byUser))

	for i := range byUser {
		all[i] = imur.seshes[byUser[i]]
	}

	all = util.SortBy(all, func(l, r dao.Session) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemorySessionsRepository) GetAllByGame(ctx context.Context, id uuid.UUID) ([]dao.Session, error) {
	byGame := imur.byGameIDIndex[id]
	if len(byGame) < 1 {
		return nil, dao.ErrNotFound
	}

	all := make([]dao.Session, len(byGame))

	for i := range byGame {
		all[i] = imur.seshes[byGame[i]]
	}

	all = util.SortBy(all, func(l, r dao.Session) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemorySessionsRepository) Update(ctx context.Context, id uuid.UUID, s dao.Session) (dao.Session, error) {
	existing, ok := imur.seshes[id]
	if !ok {
		return dao.Session{}, dao.ErrNotFound
	}

	// check for conflicts on this table only
	// (inmem does not support enforcement of foreign keys)
	if s.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.seshes[s.ID]; ok {
			return dao.Session{}, dao.ErrConstraintViolation
		}
	}

	imur.seshes[s.ID] = s
	if s.ID != id {
		delete(imur.seshes, id)

		// also update it in the index slices if we are not about to remove it
		if existing.UserID == s.UserID {
			byUser := imur.byUserIDIndex[existing.UserID]
			pos := util.SliceIndexOf(id, byUser)
			if pos < 0 {
				return dao.Session{}, fmt.Errorf("DB ASSERTION FAILURE: missing index entry for user %s to sesh %s", existing.UserID, existing.ID)
			}
			byUser[pos] = s.ID
			imur.byUserIDIndex[existing.UserID] = byUser
		}
		if existing.GameID == s.GameID {
			byGame := imur.byGameIDIndex[existing.GameID]
			pos := util.SliceIndexOf(id, byGame)
			if pos < 0 {
				return dao.Session{}, fmt.Errorf("DB ASSERTION FAILURE: missing index entry for game %s to sesh %s", existing.GameID, existing.ID)
			}
			byGame[pos] = s.ID
			imur.byGameIDIndex[existing.GameID] = byGame
		}
	}

	if s.UserID != existing.UserID {
		// if we're modifying the user, we must remove it from old index
		// entry and put it into another.
		byUser := imur.byUserIDIndex[existing.UserID]
		updated := util.SliceRemove(existing.ID, byUser)
		imur.byUserIDIndex[existing.UserID] = updated
		if len(updated) < 1 {
			delete(imur.byUserIDIndex, existing.UserID)
		}

		newByUser := imur.byUserIDIndex[s.UserID]
		newByUser = append(newByUser, s.ID)
		imur.byUserIDIndex[s.UserID] = newByUser
	}

	if s.GameID != existing.GameID {
		// if we're modifying the game, we must remove it from old index
		// entry and put it into another.
		byGame := imur.byGameIDIndex[existing.GameID]
		updated := util.SliceRemove(existing.ID, byGame)
		imur.byGameIDIndex[existing.GameID] = updated
		if len(updated) < 1 {
			delete(imur.byGameIDIndex, existing.GameID)
		}

		newByGame := imur.byGameIDIndex[s.GameID]
		newByGame = append(newByGame, s.ID)
		imur.byGameIDIndex[s.GameID] = newByGame
	}

	return s, nil
}

func (imur *InMemorySessionsRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.Session, error) {
	s, ok := imur.seshes[id]
	if !ok {
		return dao.Session{}, dao.ErrNotFound
	}

	return s, nil
}

func (imur *InMemorySessionsRepository) Delete(ctx context.Context, id uuid.UUID) (dao.Session, error) {
	s, ok := imur.seshes[id]
	if !ok {
		return dao.Session{}, dao.ErrNotFound
	}

	byUser := imur.byUserIDIndex[s.UserID]
	userUpdated := util.SliceRemove(s.ID, byUser)
	imur.byUserIDIndex[s.UserID] = userUpdated
	if len(userUpdated) < 1 {
		delete(imur.byUserIDIndex, s.UserID)
	}

	byGame := imur.byGameIDIndex[s.GameID]
	gameUpdated := util.SliceRemove(s.ID, byGame)
	imur.byGameIDIndex[s.GameID] = gameUpdated
	if len(gameUpdated) < 1 {
		delete(imur.byGameIDIndex, s.GameID)
	}

	delete(imur.seshes, s.ID)

	return s, nil
}
