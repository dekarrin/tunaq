package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/google/uuid"
	"modernc.org/sqlite"
)

func NewUsersDBConn(file string) (*UsersDB, error) {
	repo := &UsersDB{}

	var err error
	repo.db, err = sql.Open("sqlite", file)
	if err != nil {
		return nil, err
	}

	_, err = repo.db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT NOT NULL PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role INTEGER NOT NULL,
		email TEXT NOT NULL,
		last_logout_time INTEGER NOT NULL
	);`)
	if err != nil {
		return nil, wrapDBError(err)
	}

	return repo, nil
}

type UsersDB struct {
	db *sql.DB
}

func (repo *UsersDB) Create(ctx context.Context, user dao.User) (dao.User, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return dao.User{}, fmt.Errorf("could not generate ID: %w", err)
	}

	stmt, err := repo.db.Prepare(`INSERT INTO users (id, username, password, role, email, last_logout_time) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return dao.User{}, wrapDBError(err)
	}
	newEmail := ""
	if user.Email != nil {
		newEmail = user.Email.Address
	}
	_, err = stmt.ExecContext(ctx, newUUID.String(), user.Username, user.Password, user.Role.String(), newEmail, user.LastLogoutTime.Unix())
	if err != nil {
		return dao.User{}, wrapDBError(err)
	}

	return repo.GetByID(ctx, newUUID)
}

func (repo *UsersDB) GetAll(ctx context.Context) ([]dao.User, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT id, username, password, role, email, last_logout_time FROM users;`)
	if err != nil {
		return nil, wrapDBError(err)
	}
	defer rows.Close()

	var all []dao.User

	for rows.Next() {
		var user dao.User
		var email string
		var logoutTime int
		var role string
		var id string
		err = rows.Scan(
			&id,
			&user.Username,
			&user.Password,
			&role,
			&email,
			&logoutTime,
		)

		if err != nil {
			return nil, wrapDBError(err)
		}

		user.ID, err = uuid.Parse(id)
		if err != nil {
			return all, fmt.Errorf("stored UUID %q is invalid", id)
		}
		if email != "" {
			user.Email, err = mail.ParseAddress(email)
			if err != nil {
				return all, fmt.Errorf("stored email %q is invalid: %w", email, err)
			}
		}
		user.LastLogoutTime = time.Unix(int64(logoutTime), 0)
		user.Role, err = dao.ParseRole(role)
		if err != nil {
			return all, fmt.Errorf("stored role %q is invalid: %w", role, err)
		}

		all = append(all, user)
	}

	if err := rows.Err(); err != nil {
		return all, wrapDBError(err)
	}

	return all, nil
}

func (repo *UsersDB) Update(ctx context.Context, id uuid.UUID, user dao.User) (dao.User, error) {
	newEmail := ""
	if user.Email != nil {
		newEmail = user.Email.Address
	}
	res, err := repo.db.ExecContext(ctx, `UPDATE users SET id=?, username=?, password=?, role=?, email=?, last_logout_time=? WHERE id=?;`,
		user.ID.String(),
		user.Username,
		user.Password,
		user.Role.String(),
		newEmail,
		user.LastLogoutTime.Unix(),
		id.String(),
	)
	if err != nil {
		return dao.User{}, wrapDBError(err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return dao.User{}, wrapDBError(err)
	}
	if rowsAff < 1 {
		return dao.User{}, dao.ErrNotFound
	}

	return repo.GetByID(ctx, user.ID)
}

func (repo *UsersDB) GetByUsername(ctx context.Context, username string) (dao.User, error) {
	user := dao.User{
		Username: username,
	}
	var id string
	var role string
	var email string
	var logout int

	row := repo.db.QueryRowContext(ctx, `SELECT id, password, role, email, last_logout_time FROM users WHERE username = ?;`,
		username,
	)
	err := row.Scan(
		&id,
		&user.Password,
		&role,
		&email,
		&logout,
	)

	if err != nil {
		return user, wrapDBError(err)
	}

	user.ID, err = uuid.Parse(id)
	if err != nil {
		return user, fmt.Errorf("stored UUID %q is invalid", id)
	}

	if email != "" {
		user.Email, err = mail.ParseAddress(email)
		if err != nil {
			return user, fmt.Errorf("stored email %q is invalid: %w", email, err)
		}
	}
	user.LastLogoutTime = time.Unix(int64(logout), 0)
	user.Role, err = dao.ParseRole(role)
	if err != nil {
		return user, fmt.Errorf("stored role %q is invalid: %w", role, err)

	}

	return user, nil
}
func (repo *UsersDB) GetByID(ctx context.Context, id uuid.UUID) (dao.User, error) {
	user := dao.User{
		ID: id,
	}
	var role string
	var email string
	var logout int

	row := repo.db.QueryRowContext(ctx, `SELECT username, password, role, email, last_logout_time FROM users WHERE id = ?;`,
		id.String(),
	)
	err := row.Scan(
		&user.Username,
		&user.Password,
		&role,
		&email,
		&logout,
	)

	if err != nil {
		return user, wrapDBError(err)
	}

	if email != "" {
		user.Email, err = mail.ParseAddress(email)
		if err != nil {
			return user, fmt.Errorf("stored email %q is invalid: %w", email, err)
		}
	}
	user.LastLogoutTime = time.Unix(int64(logout), 0)
	user.Role, err = dao.ParseRole(role)
	if err != nil {
		return user, fmt.Errorf("stored role %q is invalid: %w", role, err)

	}

	return user, nil
}

func (repo *UsersDB) Delete(ctx context.Context, id uuid.UUID) (dao.User, error) {
	curVal, err := repo.GetByID(ctx, id)
	if err != nil {
		return curVal, err
	}

	res, err := repo.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id.String())
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

func (repo *UsersDB) Close() error {
	return repo.db.Close()
}

func wrapDBError(err error) error {
	sqliteErr := &sqlite.Error{}
	if errors.As(err, &sqliteErr) {
		if sqliteErr.Code() == 19 {
			return dao.ErrConstraintViolation
		}
		return fmt.Errorf("%s", sqlite.ErrorCodeString[sqliteErr.Code()])
	} else if errors.Is(err, sql.ErrNoRows) {
		return dao.ErrNotFound
	}
	return err
}
