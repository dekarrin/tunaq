package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
)

func NewRegistrationsDBConn(file string) (*RegistrationsDB, error) {
	repo := &RegistrationsDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, repo.init(false)
}

type RegistrationsDB struct {
	db *sql.DB
}

func (repo *RegistrationsDB) init(fk bool) error {
	// FKs not possible due to separate table files.
	stmt := `CREATE TABLE IF NOT EXISTS registrations (
		id TEXT NOT NULL PRIMARY KEY,
		user_id TEXT NOT NULL`

	if fk {
		stmt += ` REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE`
	}

	stmt += `,
		code TEXT NOT NULL,
		created INTEGER NOT NULL,
		expires INTEGER NOT NULL
	);`
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return wrapDBError(err)
	}
	return nil
}

func (repo *RegistrationsDB) Create(ctx context.Context, reg dao.Registration) (dao.Registration, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.Registration{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO registrations (id, user_id, code, created, expires) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return dao.Registration{}, wrapDBError(err)
	}
	now := time.Now()
	if reg.Expires.IsZero() {
		reg.Expires = now.Add(time.Hour)
	}
	_, err = stmt.ExecContext(
		ctx,
		convertToDB_UUID(newUUID),
		convertToDB_UUID(reg.UserID),
		reg.Code,
		convertToDB_Time(now),
		convertToDB_Time(reg.Expires),
	)
	if err != nil {
		return dao.Registration{}, wrapDBError(err)
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

		err = convertFromDB_UUID(id, &reg.ID)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid: %w", id, err)
		}
		err = convertFromDB_UUID(userID, &reg.UserID)
		if err != nil {
			return all, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
		}
		err = convertFromDB_Time(created, &reg.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}
		err = convertFromDB_Time(expires, &reg.Expires)
		if err != nil {
			return all, fmt.Errorf("stored expiration time %d is invalid: %w", expires, err)
		}

		all = append(all, reg)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *RegistrationsDB) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]dao.Registration, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, code, created, expires FROM registrations WHERE user_id=?;`,
		convertToDB_UUID(userID),
	)
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

		err = convertFromDB_UUID(id, &reg.ID)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid: %w", id, err)
		}
		err = convertFromDB_Time(created, &reg.Created)
		if err != nil {
			return all, fmt.Errorf("stored created time %d is invalid: %w", created, err)
		}
		err = convertFromDB_Time(expires, &reg.Expires)
		if err != nil {
			return all, fmt.Errorf("stored expiration time %d is invalid: %w", expires, err)
		}

		all = append(all, reg)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *RegistrationsDB) Update(ctx context.Context, id uuid.UUID, reg dao.Registration) (dao.Registration, error) {
	res, err := repo.db.ExecContext(ctx, `UPDATE registrations SET id=?, user_id=?, code=?, created=?, expires=? WHERE id=?;`,
		convertToDB_UUID(reg.ID),
		convertToDB_UUID(reg.UserID),
		reg.Code,
		convertToDB_Time(reg.Created),
		convertToDB_Time(reg.Expires),
		convertToDB_UUID(id),
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

func (repo *RegistrationsDB) GetByID(ctx context.Context, id uuid.UUID) (dao.Registration, error) {
	reg := dao.Registration{
		ID: id,
	}
	var userID string
	var created int64
	var expires int64

	row := repo.db.QueryRowContext(ctx, `SELECT user_id, code, created, expires FROM registrations WHERE id = ?;`,
		convertToDB_UUID(id),
	)
	err := row.Scan(
		&userID,
		&reg.Code,
		&created,
		&expires,
	)

	if err != nil {
		return reg, wrapDBError(err)
	}

	err = convertFromDB_UUID(userID, &reg.UserID)
	if err != nil {
		return reg, fmt.Errorf("stored user ID %q is invalid: %w", userID, err)
	}
	err = convertFromDB_Time(created, &reg.Created)
	if err != nil {
		return reg, fmt.Errorf("stored created time %d is invalid: %w", created, err)
	}
	err = convertFromDB_Time(expires, &reg.Expires)
	if err != nil {
		return reg, fmt.Errorf("stored expiration time %d is invalid: %w", expires, err)
	}

	return reg, nil
}

func (repo *RegistrationsDB) Delete(ctx context.Context, id uuid.UUID) (dao.Registration, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM registrations WHERE id = ?`, convertToDB_UUID(id))
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
