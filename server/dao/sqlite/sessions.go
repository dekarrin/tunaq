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

func (repo *RegistrationsDB) GetAll(ctx context.Context) ([]dao.Registration, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, user_id, code, created, expires FROM registrations;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Registration

	for rows.Next() {
		var reg dao.Registration
		var id string
		var userID string
		var created int64
		var expires int64
		err = rows.Scan(
			&id,
			&userID,
			&reg.Code,
			&created,
			&expires,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		reg.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid", id)
		}
		reg.UserID, err = uuid.Parse(userID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
		}
		reg.Created = time.Unix(created, 0)
		reg.Expires = time.Unix(expires, 0)

		all = append(all, reg)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *RegistrationsDB) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]dao.Registration, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, code, created, expires FROM registrations WHERE user_id=?;`, userID.String())
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.Registration

	for rows.Next() {
		reg := dao.Registration{
			UserID: userID,
		}
		var id string
		var created int64
		var expires int64
		err = rows.Scan(
			&id,
			&reg.Code,
			&created,
			&expires,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		reg.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid", id)
		}
		reg.Created = time.Unix(created, 0)
		reg.Expires = time.Unix(expires, 0)

		all = append(all, reg)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *RegistrationsDB) Update(ctx context.Context, id uuid.UUID, reg dao.Registration) (dao.Registration, error) {
	res, err := repo.db.ExecContext(ctx, `UPDATE registrations SET id=?, user_id=?, code=?, created=?, expires=? WHERE id=?;`,
		reg.ID.String(),
		reg.UserID.String(),
		reg.Code,
		reg.Created.Unix(),
		reg.Expires.Unix(),
		id.String(),
	)
	if err != nil {
		return dao.Registration{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.Registration{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.Registration{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, reg.ID)
}

func (repo *SessionsDB) GetByID(ctx context.Context, id uuid.UUID) (dao.Session, error) {
	s := dao.Session{
		ID: id,
	}
	var userID string
	var gameID string
	var created int64
	var encState string

	row := repo.db.QueryRowContext(ctx, `SELECT user_id, game_id, created, state FROM sessions WHERE id = ?;`,
		id.String(),
	)
	err := row.Scan(
		&userID,
		&gameID,
		&created,
		&encState,
	)

	if err != nil {
		return s, wrapDBError(err)
	}

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
		return s, fmt.Errorf("stored game state for %s is invalid: %w", s.ID.String(), stateData)
	}
	s.Expires = time.Unix(expires, 0)

	return s, nil
}

func (repo *RegistrationsDB) Delete(ctx context.Context, id uuid.UUID) (dao.Registration, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM registrations WHERE id = ?`, id.String())
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

func (repo *RegistrationsDB) Close() error {
	return repo.db.Close()
}
