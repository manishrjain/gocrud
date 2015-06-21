package store

import "github.com/crud/x"

type Store interface {
	Init(string)
	Commit(string, x.Instruction) bool
	IsNew(string, string) bool
	GetEntity(string, string) ([]x.Instruction, error)
	// ReadEntity(string, string) (x.Node, error)
}
