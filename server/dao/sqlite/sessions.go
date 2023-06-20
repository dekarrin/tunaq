package sqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/dekarrin/rezi"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewSessionsDBConn(file string) (*SessionsDB, error) {
	repo := &SessionsDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, repo.init(false)
}

type SessionsDB struct {
	db *sql.DB
}

func (repo *SessionsDB) init(fk bool) error {
	// FKs not possible due to separate table files.
	stmt := `CREATE TABLE IF NOT EXISTS sessions (
		id TEXT NOT NULL PRIMARY KEY,
		user_id TEXT NOT NULL`

	if fk {
		stmt += ` REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE`
	}

	stmt += `,
		game_id TEXT NOT NULL`

	if fk {
		stmt += ` REFERENCES games(id) ON DELETE CASCADE ON UPDATE CASCADE`
	}

	stmt += `,
		state TEXT NOT NULL,
		created INTEGER NOT NULL
	);`
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return wrapDBError(err)
	}
	return nil
}

func (repo *SessionsDB) Create(ctx context.Context, s dao.Session) (dao.Session, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Session{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO sessions (id, user_id, game_id, state, created) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return dao.Session{}, wrapDBError(err)
	}
	now := time.Now()

	stateData := rezi.EncBinary(s.State)
	encState := base64.StdEncoding.EncodeToString(stateData)
	_, err = stmt.ExecContext(ctx, newUUID.String(), s.UserID, s.GameID, encState, now.Unix())
	if err != nil {
		return dao.Session{}, wrapDBError(err)
	}

	return repo.GetByID(ctx, newUUID)
}

func (repo *SessionsDB) GetAll(ctx context.Context) ([]dao.Session, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, user_id, game_id, state, created FROM sessions;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Session

	for rows.Next() {
		var s dao.Session
		var id string
		var userID string
		var gameID string
		var encState string
		var created int64
		err = rows.Scan(
			&id,
			&userID,
			&gameID,
			&encState,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		s.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		s.UserID, err = uuid.Parse(userID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
		}
		s.GameID, err = uuid.Parse(gameID)
		if err != nil {
			return all, fmt.Errorf("stored game ID %q is invalid: %w", gameID, err)
		}

		s.Created = time.Unix(created, 0)

		stateData, err := base64.StdEncoding.DecodeString(encState)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: base64 decode: %w", s.ID.String(), stateData)
		}

		n, err := rezi.DecBinary(stateData, s.State)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: rezi decode: %w", s.ID.String(), err)
		}
		if n != len(stateData) {
			return all, fmt.Errorf("stored game state for %s is invalid: decoded byte count mismatch; only consumed %d/%d bytes", s.ID.String(), n, len(stateData))
		}

		all = append(all, s)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *SessionsDB) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]dao.Session, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, game_id, state, created FROM sessions WHERE user_id=?;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Session

	for rows.Next() {
		s := dao.Session{
			UserID: userID,
		}
		var id string
		var gameID string
		var encState string
		var created int64
		err = rows.Scan(
			&id,
			&gameID,
			&encState,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		s.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		s.GameID, err = uuid.Parse(gameID)
		if err != nil {
			return all, fmt.Errorf("stored game ID %q is invalid: %w", gameID, err)
		}

		s.Created = time.Unix(created, 0)

		stateData, err := base64.StdEncoding.DecodeString(encState)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: base64 decode: %w", s.ID.String(), stateData)
		}

		n, err := rezi.DecBinary(stateData, s.State)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: rezi decode: %w", s.ID.String(), err)
		}
		if n != len(stateData) {
			return all, fmt.Errorf("stored game state for %s is invalid: decoded byte count mismatch; only consumed %d/%d bytes", s.ID.String(), n, len(stateData))
		}

		all = append(all, s)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *SessionsDB) GetAllByGame(ctx context.Context, gameID uuid.UUID) ([]dao.Session, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, user_id, state, created FROM sessions;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Session

	for rows.Next() {
		s := dao.Session{
			GameID: gameID,
		}
		var id string
		var userID string
		var encState string
		var created int64
		err = rows.Scan(
			&id,
			&userID,
			&encState,
			&created,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		s.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored ID %q is invalid: %w", id, err)
		}
		s.UserID, err = uuid.Parse(userID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
		}

		s.Created = time.Unix(created, 0)

		stateData, err := base64.StdEncoding.DecodeString(encState)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: base64 decode: %w", s.ID.String(), stateData)
		}

		n, err := rezi.DecBinary(stateData, s.State)
		if err != nil {
			return all, fmt.Errorf("stored game state for %s is invalid: rezi decode: %w", s.ID.String(), err)
		}
		if n != len(stateData) {
			return all, fmt.Errorf("stored game state for %s is invalid: decoded byte count mismatch; only consumed %d/%d bytes", s.ID.String(), n, len(stateData))
		}

		all = append(all, s)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *SessionsDB) Update(ctx context.Context, id uuid.UUID, s dao.Session) (dao.Session, error) {
	// TODO: check all to ensure that 'Created' remains a dao-enforced constant
	// and that nothing is allowed to update it.

	stateData := rezi.EncBinary(s.State)
	encState := base64.StdEncoding.EncodeToString(stateData)

	res, err := repo.db.ExecContext(ctx, `UPDATE sessions SET id=?, user_id=?, game_id=?, state=? WHERE id=?;`,
		s.ID.String(),
		s.UserID.String(),
		s.GameID.String(),
		encState,
		id.String(),
	)
	if err != nil {
		return dao.Session{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.Session{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.Session{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, s.ID)
}

func (repo *SessionsDB) GetByID(ctx context.Context, id uuid.UUID) (dao.Session, error) {
	s := dao.Session{
		ID: id,
	}
	var userID string
	var gameID string
	var encState string
	var created int64

	row := repo.db.QueryRowContext(ctx, `SELECT user_id, game_id, state, created FROM sessions WHERE id = ?;`,
		id.String(),
	)
	err := row.Scan(
		&userID,
		&gameID,
		&encState,
		&created,
	)

	if err != nil {
		return s, wrapDBError(err)
	}

	// TODO: move these and all other 'stored X is invalid' errors to use serr
	// API and specifically include cause ErrDecoding
	s.UserID, err = uuid.Parse(userID)
	if err != nil {
		return s, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
	}
	s.GameID, err = uuid.Parse(gameID)
	if err != nil {
		return s, fmt.Errorf("stored game ID %q is invalid: %w", gameID, err)
	}

	s.Created = time.Unix(created, 0)

	stateData, err := base64.StdEncoding.DecodeString(encState)
	if err != nil {
		return s, fmt.Errorf("stored game state for %s is invalid: base64 decode: %w", s.ID.String(), stateData)
	}

	n, err := rezi.DecBinary(stateData, s.State)
	if err != nil {
		return s, fmt.Errorf("stored game state for %s is invalid: rezi decode: %w", s.ID.String(), err)
	}
	if n != len(stateData) {
		return s, fmt.Errorf("stored game state for %s is invalid: decoded byte count mismatch; only consumed %d/%d bytes", s.ID.String(), n, len(stateData))
	}

	return s, nil
}

func (repo *SessionsDB) Delete(ctx context.Context, id uuid.UUID) (dao.Session, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id.String())
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

func (repo *SessionsDB) Close() error {
	return repo.db.Close()
}
