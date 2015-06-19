package store

import "github.com/crud/x"

type Store interface {
	Init(string)
	Commit(string, x.Instruction) bool
	ReadEntity(string, string) Node
}
