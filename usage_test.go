package usage_test

import (
	"fmt"
	"io/ioutil"
	"time"

	_ "gopkg.in/manishrjain/gocrud.v1/drivers/leveldb"
	_ "gopkg.in/manishrjain/gocrud.v1/drivers/memsearch"
	"gopkg.in/manishrjain/gocrud.v1/indexer"
	"gopkg.in/manishrjain/gocrud.v1/req"
	"gopkg.in/manishrjain/gocrud.v1/search"
	"gopkg.in/manishrjain/gocrud.v1/store"
	"gopkg.in/manishrjain/gocrud.v1/x"
)

var log = x.Log("usage")

func ExampleStore() {
	path, err := ioutil.TempDir("", "gocrudldb_")
	if err != nil {
		x.LogErr(log, err).Fatal("Opening file")
		return
	}
	store.Get().Init(path) // leveldb

	// Update some data.
	c := req.NewContext(10) // 62^10 permutations
	err = store.NewUpdate("Root", "bigbang").SetSource("author").
		Set("when", "13.8 billion years ago").Set("explosive", true).Execute(c)
	if err != nil {
		x.LogErr(log, err).Fatal("Commiting update")
		return
	}

	// Retrieve that data
	result, err := store.NewQuery("bigbang").Run()
	if err != nil {
		x.LogErr(log, err).Fatal("While querying store")
		return
	}
	fmt.Println(result.Kind) // Root
	fmt.Println(result.Id)   // bigbang

	data := result.ToMap()
	{
		val, ok := data["explosive"]
		if !ok {
			log.Fatal("creator should be set")
			return
		}
		fmt.Println(val) // true
	}
	{
		val, ok := data["when"]
		if !ok {
			log.Fatal("creator should be set")
			return
		}
		fmt.Println(val)
	}
	// Output:
	// Root
	// bigbang
	// true
	// 13.8 billion years ago
}

type SimpleIndexer struct {
}

func (si SimpleIndexer) OnUpdate(e x.Entity) (result []x.Entity) {
	result = append(result, e)
	return
}

func (si SimpleIndexer) Regenerate(e x.Entity) (rdoc x.Doc) {
	rdoc.Id = e.Id
	rdoc.Kind = e.Kind
	rdoc.NanoTs = time.Now().UnixNano()

	result, err := store.NewQuery(e.Id).Run()
	if err != nil {
		x.LogErr(log, err).Fatal("While querying store")
		return
	}
	data := result.ToMap()
	rdoc.Data = data
	return
}

var particles = [...]string{
	"up", "charm", "top", "gluon", "down", "strange",
	"bottom", "photon", "boson", "higgs boson",
}

func ExampleSearch() {
	path, err := ioutil.TempDir("", "gocrudldb_")
	if err != nil {
		x.LogErr(log, err).Fatal("Opening file")
		return
	}
	store.Get().Init(path) // leveldb
	search.Get().Init()    // memsearch

	// Run indexer to update entities in search engine in real time.
	c := req.NewContextWithUpdates(10, 100)
	indexer.Register("Child", SimpleIndexer{})
	indexer.Run(c, 2)

	u := store.NewUpdate("Root", "bigbang").SetSource("author")
	for i := 0; i < 10; i++ {
		child := u.AddChild("Child").Set("pos", i).Set("particle", particles[i])
		if i == 5 {
			child.MarkDeleted() // This shouldn't be retrieved anymore.
		}
	}
	if err = u.Execute(c); err != nil {
		x.LogErr(log, err).Fatal("While updating")
		return
	}

	indexer.WaitForDone(c) // Block until indexing is done.

	docs, err := search.Get().NewQuery("Child").Order("-data.pos").Run()
	if err != nil {
		x.LogErr(log, err).Fatal("While searching")
		return
	}
	fmt.Println("docs:", len(docs))
	for _, doc := range docs {
		m := doc.Data.(map[string]interface{})
		fmt.Println(m["pos"], m["particle"])
	}

	// Output:
	// docs: 9
	// 9 higgs boson
	// 8 boson
	// 7 photon
	// 6 bottom
	// 4 down
	// 3 gluon
	// 2 top
	// 1 charm
	// 0 up
}
