package inmem

import (
	"fmt"

	"github.com/dekarrin/tunaq/server/dao"
)

type store struct {
	users *InMemoryUsersRepository
	regs  *InMemoryRegistrationsRepository
}

func NewDatastore() dao.Store {
	return &store{
		users: NewUsersRepository(),
		regs:  NewRegistrationsRepository(),
	}
}

func (s *store) Users() dao.UserRepository {
	return s.users
}

func (s *store) Registrations() dao.RegistrationRepository {
	return s.regs
}

func (s *store) Close() error {
	var err error
	var nextErr error

	nextErr = s.users.Close()
	if nextErr != err {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally, %w", err, nextErr)
		}
	}
	nextErr = s.regs.Close()
	if nextErr != err {
		if err != nil {
			err = fmt.Errorf("%s\nadditionally, %w", err, nextErr)
		}
	}

	return err
}
