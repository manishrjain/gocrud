package store

import "github.com/crud/x"

type Store interface {
	Init(string)
	Commit(string, x.Instruction) bool
	IsNew(string, string, string) bool
	// ReadEntity(string, string) (x.Node, error)
}
