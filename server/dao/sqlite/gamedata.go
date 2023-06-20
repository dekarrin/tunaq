package sqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewGameDatasDBConn(file string) (*GameDatasDB, error) {
	repo := &GameDatasDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, repo.init()
}

type GameDatasDB struct {
	db *sql.DB
}

func (repo *GameDatasDB) init() error {
	_, err := repo.db.Exec(`CREATE TABLE IF NOT EXISTS gamedata (
		id TEXT NOT NULL PRIMARY KEY,
		data TEXT NOT NULL
	);`)
	if err != nil {
		return wrapDBError(err)
	}

	return nil
}

func (repo *GameDatasDB) Create(ctx context.Context, gd dao.GameData) (dao.GameData, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.GameData{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO gamedata (id, data) VALUES (?, ?)`)
	if err != nil {
		return dao.GameData{}, wrapDBError(err)
	}

	_, err = stmt.ExecContext(ctx, newUUID.String(), base64.StdEncoding.EncodeToString(gd.Data))
	if err != nil {
		return dao.GameData{}, wrapDBError(err)
	}

	return repo.GetByID(ctx, newUUID)
}

func (repo *GameDatasDB) Update(ctx context.Context, id uuid.UUID, gd dao.GameData) (dao.GameData, error) {
	res, err := repo.db.ExecContext(ctx, `UPDATE gamedata SET id=?, data=? WHERE id=?;`,
		convertToDB_UUID(gd.ID),
		convertToDB_ByteSlice(gd.Data),
		convertToDB_UUID(id),
	)
	if err != nil {
		return dao.GameData{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.GameData{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.GameData{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, gd.ID)
}

func (repo *GameDatasDB) GetByID(ctx context.Context, id uuid.UUID) (dao.GameData, error) {
	gd := dao.GameData{
		ID: id,
	}
	var data string

	row := repo.db.QueryRowContext(ctx, `SELECT data FROM gamedata WHERE id = ?;`,
		convertToDB_UUID(id),
	)
	err := row.Scan(
		&data,
	)

	if err != nil {
		return gd, wrapDBError(err)
	}

	err = convertFromDB_ByteSlice(data, &gd.Data)
	if err != nil {
		return gd, fmt.Errorf("stored data for %s is invalid: %w", gd.ID.String(), err)
	}

	return gd, nil
}

func (repo *GameDatasDB) Delete(ctx context.Context, id uuid.UUID) (dao.GameData, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM gamedata WHERE id = ?`, convertToDB_UUID(id))
	if err != nil {
		return curVal, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return curVal, wrapDBError(err)
	}
	if rowsAff < 1 {
		return curVal, dao.ErrNotFound
	}

	return curVal, nil
}

func (repo *GameDatasDB) Close() error {
	return repo.db.Close()
}
