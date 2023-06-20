package inmem

import (
	"context"
	"fmt"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewGameDatasRepository() *InMemoryGameDatasRepository {
	return &InMemoryGameDatasRepository{
		datas: make(map[uuid.UUID]dao.GameData),
	}
}

type InMemoryGameDatasRepository struct {
	datas map[uuid.UUID]dao.GameData
}

func (imur *InMemoryGameDatasRepository) Close() error {
	return nil
}

func (imur *InMemoryGameDatasRepository) Create(ctx context.Context, gd dao.GameData) (dao.GameData, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.GameData{}, fmt.Errorf("could not generate ID: %w", err)
	}

	gd.ID = newUUID

	imur.datas[gd.ID] = gd

	return gd, nil
}

func (imur *InMemoryGameDatasRepository) Update(ctx context.Context, id uuid.UUID, gd dao.GameData) (dao.GameData, error) {
	_, ok := imur.datas[id]
	if !ok {
		return dao.GameData{}, dao.ErrNotFound
	}

	if gd.ID != id {
		// that's okay but we need to check it
		if _, ok := imur.datas[gd.ID]; ok {
			return dao.GameData{}, dao.ErrConstraintViolation
		}
	}

	imur.datas[gd.ID] = gd
	if gd.ID != id {
		delete(imur.datas, id)
	}

	return gd, nil
}

func (imur *InMemoryGameDatasRepository) GetByID(ctx context.Context, id uuid.UUID) (dao.GameData, error) {
	user, ok := imur.datas[id]
	if !ok {
		return dao.GameData{}, dao.ErrNotFound
	}

	return user, nil
}

func (imur *InMemoryGameDatasRepository) Delete(ctx context.Context, id uuid.UUID) (dao.GameData, error) {
	user, ok := imur.datas[id]
	if !ok {
		return dao.GameData{}, dao.ErrNotFound
	}

	delete(imur.datas, user.ID)

	return user, nil
}
