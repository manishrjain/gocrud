package main_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"

	_ "github.com/manishrjain/gocrud/drivers/leveldb"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("usage")

func ExampleStore() {
	rand.Seed(0) // For determinism.
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
