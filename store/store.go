package store

import "github.com/crud/x"

var log = x.Log("store")

type Store interface {
	Init(string)
	Commit(tablePrefix string, its []*x.Instruction) error
	IsNew(tablePrefix string, subject string) bool
	GetEntity(tablePrefix string, subject string) ([]x.Instruction, error)
}
