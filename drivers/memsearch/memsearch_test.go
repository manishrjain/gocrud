package memsearch

import (
	"testing"

	"github.com/manishrjain/gocrud/testx"
)

func initialize(t *testing.T) *MemSearch {
	ms := new(MemSearch)
	ms.Init()

	testx.AddDocs(ms, t)
	return ms
}

func TestNewAndFilter(t *testing.T) {
	ms := initialize(t)
	testx.RunAndFilter(ms, t)
}

var soln = [...]string{
	"m81",
	"ngc 3370",
	"galaxy ngc 1512",
	"ngc 123",
	"whirlpool galaxy",
	"sombrero galaxy",
}

func TestNewOrFilter(t *testing.T) {
	ms := initialize(t)
	testx.RunOrFilter(ms, t)
}
