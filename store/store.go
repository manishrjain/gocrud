// Package store provides an interface for data store operations, to
// allow for easy extensibility to support various datastores.
package store

import "github.com/manishrjain/gocrud/x"

var log = x.Log("store")

// All the data CRUD operations are run via this Store interface.
// Implement this interface to add support for a datastore.
type Store interface {
	// Init could be used to initialize anything which needs database type
	// or tablename. This should be called exactly once.
	Init(dbtype string, tablename string)

	// Commit writes the array of instructions to the data store. tablePrefix
	// string could be used to differentiate between test and production
	// stores, for e.g., in Google datastore.
	Commit(tablePrefix string, its []*x.Instruction) error

	// IsNew returns true if the subject id provided doesn't exist in the
	// store. Note that this subject id is never solely the row primary key,
	// because multiple rows can (and most surely will) have the same subject id.
	IsNew(tablePrefix string, subject string) bool

	// GetEntity retrieves all the rows for the given subject id, parses them
	// into instructions, appends them to the array, and returns it. Any error
	// encountered during these steps is also returned.
	GetEntity(tablePrefix string, subject string) ([]x.Instruction, error)
}
