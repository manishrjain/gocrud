package memsearch

import (
	"testing"

	"github.com/manishrjain/gocrud/testx"
)

func initialize() *MemSearch {
	ms := new(MemSearch)
	ms.Init()

	testx.AddDocs(ms)
	return ms
}

func TestNewAndFilter(t *testing.T) {
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
	testx.RunOrFilter(ms, t)
}

func TestCount(t *testing.T) {
	testx.RunCount(ms, t)
}

var ms *MemSearch

func init() {
	ms = initialize()
}
