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
	db          *sql.DB
	worldDataDB *sql.DB

	users *UsersDB
	regs  *RegistrationsDB
	games *GamesDB
	gd    *GameDatasDB
}

func NewDatastore(storageDir string) (dao.Store, error) {
	st := &store{}

	fileName := filepath.Join(storageDir, "data.db")
	worldFileName := filepath.Join(storageDir, "worlds.db")

	var err error
	st.db, err = sql.Open("sqlite", fileName)
	if err != nil {
		return nil, wrapDBError(err)
	}
	st.worldDataDB, err = sql.Open("sqlite", worldFileName)
	if err != nil {
		return nil, wrapDBError(err)
	}

	st.gd = &GameDatasDB{db: st.db}
	st.gd.init()

	st.users = &UsersDB{db: st.db}
	st.users.init()

	st.regs = &RegistrationsDB{db: st.db}
	st.regs.init(true)

	st.games = &GamesDB{db: st.db}
	st.games.init(true)

	return st, nil
}

func (s *store) Users() dao.UserRepository {
	return s.users
}

func (s *store) Registrations() dao.RegistrationRepository {
	return s.regs
}

func (s *store) Games() dao.GameRepository {
	return s.games
}

func (s *store) GameData() dao.GameDataRepository {
	return s.gd
}

func (s *store) Close() error {
	s.worldDataDB.Close()
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
