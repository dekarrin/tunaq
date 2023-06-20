package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/dekarrin/tunaq/server/dao"
	"modernc.org/sqlite"
)

type store struct {
	db *sql.DB

	users *UsersDB
	regs  *RegistrationsDB
}

func NewDatastore(storageDir string) (dao.Store, error) {
	fileName := filepath.Join(storageDir, "data.db")
	usersDB, err := NewUsersDBConn(fileName)
	if err != nil {
		return nil, err
	}

	st := &store{
		users: usersDB,
	}
	st.db = usersDB.db

	st.regs = &RegistrationsDB{db: st.db}
	st.regs.init(true)

	return st, nil
}

func (s *store) Users() dao.UserRepository {
	return s.users
}

func (s *store) Registrations() dao.RegistrationRepository {
	return s.regs
}

func (s *store) Close() error {
	return s.db.Close()
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
