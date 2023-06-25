// Package tunas has services for interacting with the TunaQuest Server backend
// decoupled from the API that accesses it.
package tunas

import (
	"github.com/dekarrin/tunaq/server/dao"
)

// Service is a service for interacting with and modifying the TunaQuest server
// backend. It performs the actions requested and makes calls to server
// persistence to preserve the backend state.
//
// The zero-value of Service is not ready to be used; assign a valid DAO store
// to DB before attempting to use it.
type Service struct {

	// DB is the persistence store of the service.
	DB dao.Store
}
