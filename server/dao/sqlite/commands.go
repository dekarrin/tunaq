package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewCommandsDBConn(file string) (*CommandsDB, error) {
	repo := &CommandsDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, repo.init(false)
}

type CommandsDB struct {
	db         *sql.DB
	multiTable bool
}

func (repo *CommandsDB) init(fk bool) error {
	repo.multiTable = fk

	stmt := `CREATE TABLE IF NOT EXISTS sessions (
		id TEXT NOT NULL PRIMARY KEY,
		session_id TEXT NOT NULL`

	if fk {
		stmt += ` REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE`
	}

	stmt += `,
		content TEXT NOT NULL,
		created INTEGER NOT NULL
	);`
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return wrapDBError(err)
	}
	return nil
}

func (repo *CommandsDB) Create(ctx context.Context, c dao.Command) (dao.Command, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Command{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO commands (id, session_id, content, created) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return dao.Command{}, wrapDBError(err)
	}
	now := time.Now()

	_, err = stmt.ExecContext(
		ctx,
		convertToDB_UUID(newUUID),
		convertToDB_UUID(c.SessionID),
		c.Command,
		convertToDB_Time(now),
	)
	if err != nil {
		return dao.Command{}, wrapDBError(err)
	}

	return repo.GetByID(ctx, newUUID)
}

func (repo *CommandsDB) GetAll(ctx context.Context) ([]dao.Command, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, session_id, content, created FROM commands;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Command

	for rows.Next() {
		var c dao.Command
		var id string
		var seshID string
		var created int64
		err = rows.Scan(
			&id,
			&seshID,
			&c.Command,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		err = convertFromDB_UUID(id, &c.ID)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		err = convertFromDB_UUID(seshID, &c.SessionID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", seshID, err)
		}
		err = convertFromDB_Time(created, &c.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}

		all = append(all, c)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *CommandsDB) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]dao.Command, error) {
	// this function is impossible unless it has been inited with fk support
	if !repo.multiTable {
		return nil, fmt.Errorf("cannot do cross-table join query without multi-table support")
	}

	rows, err := repo.db.QueryContext(ctx, `
		SELECT C.id, C.session_id, C.content, C.created
		FROM commands AS C
		INNER JOIN sessions AS S
			ON S.id = C.session_id
		WHERE C.user_id=?
	;`,
		convertToDB_UUID(userID),
	)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Command

	for rows.Next() {
		var c dao.Command
		var id string
		var seshID string
		var created int64
		err = rows.Scan(
			&id,
			&seshID,
			&c.Command,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		err = convertFromDB_UUID(id, &c.ID)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		err = convertFromDB_UUID(seshID, &c.SessionID)
		if err != nil {
			return all, fmt.Errorf("stored session ID %q is invalid: %w", seshID, err)
		}
		err = convertFromDB_Time(created, &c.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}

		all = append(all, c)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *CommandsDB) GetAllBySession(ctx context.Context, sessionID uuid.UUID) ([]dao.Command, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, content, created FROM commands;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Command

	for rows.Next() {
		c := dao.Command{
			SessionID: sessionID,
		}
		var id string
		var created int64
		err = rows.Scan(
			&id,
			&c.Command,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		err = convertFromDB_UUID(id, &c.ID)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		err = convertFromDB_Time(created, &c.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}

		all = append(all, c)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *CommandsDB) Update(ctx context.Context, id uuid.UUID, c dao.Command) (dao.Command, error) {
	res, err := repo.db.ExecContext(ctx, `UPDATE commands SET id=?, session_id=?, content=? WHERE id=?;`,
		convertToDB_UUID(c.ID),
		convertToDB_UUID(c.SessionID),
		c.Command,
		convertToDB_UUID(id),
	)
	if err != nil {
		return dao.Command{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.Command{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.Command{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, c.ID)
}

func (repo *CommandsDB) GetByID(ctx context.Context, id uuid.UUID) (dao.Command, error) {
	c := dao.Command{
		ID: id,
	}
	var seshID string
	var created int64

	row := repo.db.QueryRowContext(ctx, `SELECT session_id, content, created FROM commands WHERE id = ?;`,
		convertToDB_UUID(id),
	)
	err := row.Scan(
		&seshID,
		&c.Command,
		&created,
	)

	if err != nil {
		return c, wrapDBError(err)
	}

	err = convertFromDB_UUID(seshID, &c.SessionID)
	if err != nil {
		return c, fmt.Errorf("stored session ID %q is invalid: %w", seshID, err)
	}
	err = convertFromDB_Time(created, &c.Created)
	if err != nil {
		return c, fmt.Errorf("stored created time %q is invalid: %w", created, err)
	}

	return c, nil
}

func (repo *CommandsDB) Delete(ctx context.Context, id uuid.UUID) (dao.Command, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM commands WHERE id = ?`,
		convertToDB_UUID(id),
	)
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

func (repo *CommandsDB) Close() error {
	return repo.db.Close()
}
