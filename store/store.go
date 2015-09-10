// Package store provides an interface for data store operations, to
// allow for easy extensibility to support various datastores. Also, provides
// the standardized Update and Query interfaces to data stores.
package store

import (
	"gopkg.in/manishrjain/gocrud.v1/x"
)

var log = x.Log("store")

// All the data CRUD operations are run via this Store interface.
// Implement this interface to add support for a datastore.
type Store interface {
	// Init is used to initialize store driver.
	Init(args ...string)

	// Commit writes the array of instructions to the data store.
	Commit(its []*x.Instruction) error

	// IsNew returns true if the entity id provided doesn't exist in the
	// store. Note that this entity id is never solely the row primary key,
	// because multiple rows can (and most surely will) have the same entity id.
	IsNew(entityId string) bool

	// GetEntity retrieves all the rows for the given subject id, parses them
	// into instructions, appends them to the array, and returns it. Any error
	// encountered during these steps is also returned.
	GetEntity(entityId string) ([]x.Instruction, error)

	// Iterate allows for a way to page over all the entities stored in the table.
	// Iteration starts from id fromId and stops after num results are processed.
	// Note that depending upon database, number of distinct entities might be
	// less than the number of results retrieved from the store. That's normal.
	//
	// Returns the number of entities found, the last entity returned
	// and error, if any. If the number of entities found are zero, assume
	// that we've reached the end of the table.
	Iterate(fromId string, num int, ch chan x.Entity) (int, x.Entity, error)
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
