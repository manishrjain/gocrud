package indexer_test

import (
	"time"

	_ "github.com/manishrjain/gocrud/drivers/leveldb"
	_ "github.com/manishrjain/gocrud/drivers/memsearch"
	"github.com/manishrjain/gocrud/indexer"
	"github.com/manishrjain/gocrud/search"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

type SimpleIndexer struct {
}

func (si SimpleIndexer) OnUpdate(e x.Entity) (result []x.Entity) {
	result = append(result, e)
	return result
}

func (si SimpleIndexer) Regenerate(e x.Entity) (rdoc x.Doc) {
	rdoc.Id = e.Id
	rdoc.Kind = e.Kind
	rdoc.NanoTs = time.Now().UnixNano()
	return rdoc
}

func ExampleServer() {
	store.Get().Init("/tmp/ldb_" + x.UniqueString(10))
	search.Get().Init("memsearch")
	indexer.Register("EntityKind", SimpleIndexer{})

	server := indexer.NewServer(100, 5)
	server.InfiniteLoop(30 * time.Minute)
	// This would never exit.
	// OR, you could also just run this once, if you're
	// testing your setup.
	server.LoopOnce()
	server.Finish() // Finish is only useful when you're looping once.
}
