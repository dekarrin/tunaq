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
	dbFilename         string
	gameDataDBFilename string

	db         *sql.DB
	gameDataDB *sql.DB

	users  *UsersDB
	regs   *RegistrationsDB
	games  *GamesDB
	gd     *GameDatasDB
	seshes *SessionsDB
}

func NewDatastore(storageDir string) (dao.Store, error) {
	st := &store{
		dbFilename:         "data.db",
		gameDataDBFilename: "worlds.db",
	}

	fileName := filepath.Join(storageDir, st.dbFilename)
	worldFileName := filepath.Join(storageDir, st.gameDataDBFilename)

	var err error
	st.db, err = sql.Open("sqlite", fileName)
	if err != nil {
		return nil, wrapDBError(err)
	}
	st.gameDataDB, err = sql.Open("sqlite", worldFileName)
	if err != nil {
		return nil, wrapDBError(err)
	}

	st.gd = &GameDatasDB{db: st.gameDataDB}
	st.gd.init()

	st.users = &UsersDB{db: st.db}
	st.users.init()

	st.regs = &RegistrationsDB{db: st.db}
	st.regs.init(true)

	st.games = &GamesDB{db: st.db}
	st.games.init(true)

	st.seshes = &SessionsDB{db: st.db}
	st.seshes.init(true)

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

func (s *store) Sessions() dao.SessionRepository {
	return s.seshes
}

func (s *store) Close() error {
	worldsDBErr := s.gameDataDB.Close()
	mainDBErr := s.db.Close()

	var err error
	if worldsDBErr != nil {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally: %s: %w", err.Error(), s.gameDataDBFilename, worldsDBErr)
		} else {
			err = fmt.Errorf("%s: %w", s.gameDataDBFilename, worldsDBErr)
		}
	}
	if mainDBErr != nil {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally: %s: %w", err.Error(), s.dbFilename, mainDBErr)
		} else {
			err = fmt.Errorf("%s: %w", s.dbFilename, err)
		}
	}
	return err
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
