package inmem

import (
	"fmt"

	"github.com/dekarrin/tunaq/server/dao"
)

type store struct {
	users *InMemoryUsersRepository
	regs  *InMemoryRegistrationsRepository
	games *InMemoryGamesRepository
}

func NewDatastore() dao.Store {
	return &store{
		users: NewUsersRepository(),
		regs:  NewRegistrationsRepository(),
		games: NewGamesRepository(),
	}
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

func (s *store) Close() error {
	var err error
	var nextErr error

	nextErr = s.users.Close()
	if nextErr != err {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally, %w", err, nextErr)
		} else {
			err = nextErr
		}
	}
	nextErr = s.regs.Close()
	if nextErr != err {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally, %w", err, nextErr)
		} else {
			err = nextErr
		}
	}
	nextErr = s.games.Close()
	if nextErr != err {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally, %w", err, nextErr)
		} else {
			err = nextErr
		}
	}

	return err
}
