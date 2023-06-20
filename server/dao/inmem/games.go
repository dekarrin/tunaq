package inmem

import (
	"context"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/internal/util"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewGamesRepository() *InMemoryGamesRepository {
	return &InMemoryGamesRepository{
		games:         make(map[uuid.UUID]dao.Game),
		byUserIDIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

type InMemoryGamesRepository struct {
	games         map[uuid.UUID]dao.Game
	byUserIDIndex map[uuid.UUID][]uuid.UUID
}

func (imur *InMemoryGamesRepository) Close() error {
	return nil
}

func (imur *InMemoryGamesRepository) Create(ctx context.Context, g dao.Game) (dao.Game, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Game{}, fmt.Errorf("could not generate ID: %w", err)
	}

	now := time.Now()

	g.ID = newUUID
	g.Created = now
	g.Modified = now

	imur.games[g.ID] = g

	userGames := imur.byUserIDIndex[g.UserID]
	userGames = append(userGames, g.ID)
	imur.byUserIDIndex[g.UserID] = userGames

	return g, nil
}

func (imur *InMemoryGamesRepository) GetAll(ctx context.Context) ([]dao.Game, error) {
	all := make([]dao.Game, len(imur.games))

	i := 0
	for k := range imur.games {
		all[i] = imur.games[k]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Game) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemoryGamesRepository) GetAllByUser(ctx context.Context, id uuid.UUID) ([]dao.Game, error) {
	byUser := imur.byUserIDIndex[id]
	if len(byUser) < 1 {
		return nil, dao.ErrNotFound
	}

	all := make([]dao.Game, len(byUser))

	for i := range byUser {
		all[i] = imur.games[byUser[i]]
		i++
	}

	all = util.SortBy(all, func(l, r dao.Game) bool {
		return l.ID.String() < r.ID.String()
	})

	return all, nil
}

func (imur *InMemoryGamesRepository) Update(ctx context.Context, id uuid.UUID, g dao.Game) (dao.Game, error) {
	existing, ok := imur.games[id]
	if !ok {
		return dao.Game{}, dao.ErrNotFound
	}

	// check for conflicts on this table only
	// (inmem does not support enforcement of foreign keys)
	if g.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.games[g.ID]; ok {
			return dao.Game{}, dao.ErrConstraintViolation
		}
	}

	imur.games[g.ID] = g
	if g.ID != id {
		delete(imur.games, id)

		// also update it in the index slice if we are not about to remove it
		if existing.UserID == g.UserID {
			byUser := imur.byUserIDIndex[existing.UserID]
			pos := util.SliceIndexOf(id, byUser)
			if pos < 0 {
				return dao.Game{}, fmt.Errorf("DB ASSERTION FAILURE: missing index entry for user %s to game %s", existing.UserID, existing.ID)
			}
			byUser[pos] = g.ID
			imur.byUserIDIndex[existing.UserID] = byUser
		}
	}

	if g.UserID != existing.UserID {
		// if we're modifying the user, we must remove it from old index
		// entry and put it into another.
		byUser := imur.byUserIDIndex[existing.UserID]
		updated := util.SliceRemove(existing.ID, byUser)
		imur.byUserIDIndex[existing.UserID] = updated
		if len(updated) < 1 {
			delete(imur.byUserIDIndex, existing.UserID)
		}

		newByUser := imur.byUserIDIndex[g.UserID]
		newByUser = append(newByUser, g.ID)
		imur.byUserIDIndex[g.UserID] = newByUser
	}

	return g, nil
}

func (imur *InMemoryGamesRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.Game, error) {
	g, ok := imur.games[id]
	if !ok {
		return dao.Game{}, dao.ErrNotFound
	}

	return g, nil
}

func (imur *InMemoryGamesRepository) Delete(ctx context.Context, id uuid.UUID) (dao.Game, error) {
	g, ok := imur.games[id]
	if !ok {
		return dao.Game{}, dao.ErrNotFound
	}

	byUser := imur.byUserIDIndex[g.UserID]
	updated := util.SliceRemove(g.ID, byUser)
	imur.byUserIDIndex[g.UserID] = updated
	if len(updated) < 1 {
		delete(imur.byUserIDIndex, g.UserID)
	}
	delete(imur.games, g.ID)

	return g, nil
}
