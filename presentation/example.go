package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	_ "gopkg.in/manishrjain/gocrud.v1/drivers/leveldb"
	_ "gopkg.in/manishrjain/gocrud.v1/drivers/memsearch"
	"gopkg.in/manishrjain/gocrud.v1/indexer"
	"gopkg.in/manishrjain/gocrud.v1/req"
	"gopkg.in/manishrjain/gocrud.v1/search"
	"gopkg.in/manishrjain/gocrud.v1/store"
	"gopkg.in/manishrjain/gocrud.v1/x"
)

// Create a new Social Post.
func storeUpdate(ctx *req.Context) {
	// Get User entity, set author for update.
	u := store.NewUpdate("User", "userid").SetSource("userid")

	// New Post child entity for User entity.
	p := u.AddChild("Post").
		Set("url", "https://xkcd.com/936/").
		Set("body", "Password Strength") // Method chaining.
	p.AddChild("Like").Set("upvote", true)

	// Add attendees to Event entity.
	c := p.AddChild("Comment").Set("text", "Hard for humans, easy for computers.")
	c.AddChild("Like").Set("upvote", true)

	_ = u.Execute(ctx) // Returns error
}

func storeQuery() {
	// Collect all Post entities under User userid.
	p := store.NewQuery("userid").Collect("Post")
	p.Collect("Like") // Collect all Like entities for Post entities.

	// Collect all Comment entities, and Likes on those comments.
	p.Collect("Comment").Collect("Like")

	// Alternatively, get everything up to certain depth.
	p = store.NewQuery("userid").UptoDepth(10)

	result, _ := p.Run()
	js, _ := result.ToJson() // Convert to JSON.
	fmt.Println(string(js))

	// Optionally write to w http.ResponseWriter.
	// result.WriteJsonResponse(w)
}

type SimpleIndexer struct {
}

// Return all entities to be re-indexed when entity x gets updated.
func (si SimpleIndexer) OnUpdate(e x.Entity) (result []x.Entity) {
	result = append(result, e) // Add self.
	return
}

func (si SimpleIndexer) Regenerate(e x.Entity) (rdoc x.Doc) {
	rdoc.Id = e.Id
	rdoc.Kind = e.Kind
	rdoc.NanoTs = time.Now().UnixNano() // Used for versioning, to support concurrency.

	// Query store to get the entity, and all it's properties.
	result, _ := store.NewQuery(e.Id).Run()
	data := result.ToMap()
	rdoc.Data = data // Data field takes interface{}, so set to anything.
	return
} // end_simple

func searchQuery() {
	q := search.Get().NewQuery("Comment").Order("-ts_millis")
	q.NewAndFilter().
		AddRegex("text", "[hH]ard.*easy").
		AddExact("source", "userid")
	docs, _ := q.Run()
	j, _ := json.Marshal(docs)
	fmt.Println(string(j))
}

func main() {
	path, err := ioutil.TempDir("", "gocrudldb_")
	if err != nil {
		log.Fatal("Opening file")
		return
	}
	store.Get().Init(path) // leveldb
	search.Get().Init()    // memsearch

	ctx := req.NewContextWithUpdates(3, 10) // Number of chars in unique ids.
	indexer.Register("Comment", SimpleIndexer{})
	indexer.Run(ctx, 2)

	storeUpdate(ctx)
	storeQuery()

	indexer.WaitForDone(ctx)
	searchQuery()
}
