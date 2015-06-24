package store

import "github.com/crud/x"

var log = x.Log("store")

type Store interface {
	Init(string)
	Commit(string, []*x.Instruction) error
	IsNew(string, string) bool
	GetEntity(string, string) ([]x.Instruction, error)
	// ReadEntity(string, string) (x.Node, error)
}
