package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewGamesDBConn(file string) (*GamesDB, error) {
	repo := &GamesDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, repo.init(false)
}

type GamesDB struct {
	db *sql.DB
}

func (repo *GamesDB) init(fk bool) error {
	// FKs not possible due to separate table files.
	stmt := `CREATE TABLE IF NOT EXISTS games (
		id TEXT NOT NULL PRIMARY KEY,
		user_id TEXT NOT NULL`

	if fk {
		stmt += ` REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE`
	}

	stmt += `,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		description TEXT NOT NULL,
		storage TEXT NOT NULL,
		local_path TEXT NOT NULL,
		last_local_access INTEGER NOT NULL,
		created INTEGER NOT NULL,
		modified INTEGER NOT NULL
	);`
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return wrapDBError(err)
	}
	return nil
}

func (repo *GamesDB) Create(ctx context.Context, g dao.Game) (dao.Game, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Game{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO games (id, user_id, name, version, description, storage, local_path, last_local_access, created, modified) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return dao.Game{}, wrapDBError(err)
	}

	now := time.Now()

	_, err = stmt.ExecContext(
		ctx,
		convertToDB_UUID(newUUID),
		convertToDB_UUID(g.UserID),
		g.Name,
		g.Version,
		g.Description,
		g.Storage,
		g.LocalPath,
		convertToDB_Time(g.LastLocalAccess),
		convertToDB_Time(now),
		convertToDB_Time(now),
	)
	if err != nil {
		return dao.Game{}, wrapDBError(err)
	}

	return repo.GetByID(ctx, newUUID)
}

func (repo *GamesDB) GetAll(ctx context.Context) ([]dao.Game, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, user_id, name, version, description, storage, local_path, last_local_access, created, modified FROM game;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Game

	for rows.Next() {
		var g dao.Game
		var id string
		var userID string
		var created int64
		var modified int64
		var lastLocal int64
		err = rows.Scan(
			&id,
			&userID,
			&g.Name,
			&g.Version,
			&g.Description,
			&g.Storage,
			&g.LocalPath,
			&lastLocal,
			&created,
			&modified,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		err = convertFromDB_UUID(id, &g.ID)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid: %w", id, err)
		}
		err = convertFromDB_UUID(userID, &g.UserID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
		}
		err = convertFromDB_Time(created, &g.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}
		err = convertFromDB_Time(modified, &g.Modified)
		if err != nil {
			return all, fmt.Errorf("stored modified time %d is invalid: %w", modified, err)
		}
		err = convertFromDB_Time(lastLocal, &g.LastLocalAccess)
		if err != nil {
			return all, fmt.Errorf("stored last local access time %d is invalid: %w", lastLocal, err)
		}

		all = append(all, g)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *GamesDB) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]dao.Game, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, name, version, description, storage, local_path, last_local_access, created, modified FROM games WHERE user_id=?;`,
		convertToDB_UUID(userID),
	)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Game

	for rows.Next() {
		g := dao.Game{
			UserID: userID,
		}
		var id string
		var created int64
		var modified int64
		var lastLocal int64
		err = rows.Scan(
			&id,
			&g.Name,
			&g.Version,
			&g.Description,
			&g.Storage,
			&g.LocalPath,
			&lastLocal,
			&created,
			&modified,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		err = convertFromDB_UUID(id, &g.ID)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid: %w", id, err)
		}
		err = convertFromDB_Time(created, &g.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}
		err = convertFromDB_Time(modified, &g.Modified)
		if err != nil {
			return all, fmt.Errorf("stored modified time %d is invalid: %w", modified, err)
		}
		err = convertFromDB_Time(lastLocal, &g.LastLocalAccess)
		if err != nil {
			return all, fmt.Errorf("stored last local access time %d is invalid: %w", lastLocal, err)
		}

		all = append(all, g)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *GamesDB) Update(ctx context.Context, id uuid.UUID, g dao.Game) (dao.Game, error) {
	res, err := repo.db.ExecContext(ctx, `UPDATE games SET id=?, user_id=?, name=?, version=?, description=?, storage=?, local_path=?, last_local_access=?, created=?, modified=? WHERE id=?;`,
		convertToDB_UUID(g.ID),
		convertToDB_UUID(g.UserID),
		g.Name,
		g.Version,
		g.Description,
		g.Storage,
		g.LocalPath,
		convertToDB_Time(g.LastLocalAccess),
		convertToDB_Time(g.Created),
		convertToDB_Time(time.Now()),
		convertToDB_UUID(id),
	)
	if err != nil {
		return dao.Game{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.Game{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.Game{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, g.ID)
}

func (repo *GamesDB) GetByID(ctx context.Context, id uuid.UUID) (dao.Game, error) {
	g := dao.Game{
		ID: id,
	}
	var userID string
	var created int64
	var modified int64
	var lastLocal int64

	row := repo.db.QueryRowContext(ctx, `SELECT user_id, name, version, description, storage, local_path, last_local_access, created, modified FROM games WHERE id = ?;`,
		convertToDB_UUID(id),
	)
	err := row.Scan(
		&userID,
		&g.Name,
		&g.Version,
		&g.Description,
		&g.Storage,
		&g.LocalPath,
		&lastLocal,
		&created,
		&modified,
	)

	if err != nil {
		return g, wrapDBError(err)
	}

	err = convertFromDB_UUID(userID, &g.UserID)
	if err != nil {
		return g, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
	}
	err = convertFromDB_Time(created, &g.Created)
	if err != nil {
		return g, fmt.Errorf("stored created time %d is invalid: %w", created, err)
	}
	err = convertFromDB_Time(modified, &g.Modified)
	if err != nil {
		return g, fmt.Errorf("stored modified time %d is invalid: %w", modified, err)
	}
	err = convertFromDB_Time(lastLocal, &g.LastLocalAccess)
	if err != nil {
		return g, fmt.Errorf("stored last local access time %d is invalid: %w", lastLocal, err)
	}

	return g, nil
}

func (repo *GamesDB) Delete(ctx context.Context, id uuid.UUID) (dao.Game, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM games WHERE id = ?`, convertToDB_UUID(id))
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

func (repo *GamesDB) Close() error {
	return repo.db.Close()
}
