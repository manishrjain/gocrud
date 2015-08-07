// Package store provides an interface for data store operations, to
// allow for easy extensibility to support various datastores.
package store

import (
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("store")

// All the data CRUD operations are run via this Store interface.
// Implement this interface to add support for a datastore.
type Store interface {
	// Init is used to initialize store driver.
	Init(args ...string)

	// Commit writes the array of instructions to the data store.
	Commit(its []*x.Instruction) error

	// IsNew returns true if the subject id provided doesn't exist in the
	// store. Note that this subject id is never solely the row primary key,
	// because multiple rows can (and most surely will) have the same subject id.
	IsNew(subject string) bool

	// GetEntity retrieves all the rows for the given subject id, parses them
	// into instructions, appends them to the array, and returns it. Any error
	// encountered during these steps is also returned.
	GetEntity(subject string) ([]x.Instruction, error)
}

var driver Store

func Register(name string, store Store) {
	if store == nil {
		log.WithField("driver", name).Fatal("Nil store")
		return
	}
	if driver != nil {
		log.WithField("driver", name).Fatal("Register called twice")
		return
	}
	log.WithField("driver", name).Debug("Registering store driver")
	driver = store
}

func Get() Store {
	if driver == nil {
		log.Fatal("No driver registered")
		return nil
	}
	return driver
}
