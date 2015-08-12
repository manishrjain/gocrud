// Package main to demonstrate usage of the gocrud apis,
// using a social network as an example.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/search"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"

	// _ "github.com/manishrjain/gocrud/drivers/elasticsearch"
	_ "github.com/manishrjain/gocrud/drivers/leveldb"
	_ "github.com/manishrjain/gocrud/drivers/memsearch"
	// _ "github.com/manishrjain/gocrud/drivers/datastore"
	// _ "github.com/manishrjain/gocrud/drivers/sqlstore"
	// _ "github.com/manishrjain/gocrud/drivers/cassandra"
	// _ "github.com/manishrjain/gocrud/drivers/mongodb"
	// _ "github.com/manishrjain/gocrud/drivers/rethinkdb"
)

var storeType = flag.String("store", "leveldb",
	"Available stores are cass for cassandra, "+
		"sql for MySQL, leveldb for LevelDB, "+
		"datastore for Google Datastore. "+
		"LevelDB is the default.")

var log = x.Log("social")
var c *req.Context

type Like struct {
	Id string `json:"id,omitempty"`
}

type Comment struct {
	Id      string    `json:"id,omitempty"`
	Comment []Comment `json:"Comment,omitempty"`
	Like    []Like    `json:"Like,omitempty"`
}

type Post struct {
	Id      string    `json:"id,omitempty"`
	Comment []Comment `json:"Comment,omitempty"`
	Like    []Like    `json:"Like,omitempty"`
}

type User struct {
	Id   string `json:"id,omitempty"`
	Post []Post `json:"Post,omitempty"`
}

type SimpleIndexer struct {
	c *req.Context
}

func (si SimpleIndexer) OnUpdate(e x.Entity) (result []x.Entity) {
	result = append(result, e)
	return
}

func (si SimpleIndexer) Regenerate(e x.Entity) (rdoc x.Doc) {
	rdoc.Id = e.Id
	rdoc.Kind = e.Kind
	rdoc.NanoTs = int64(rand.Intn(1000))

	result, err := store.NewQuery(e.Id).UptoDepth(0).Run(c)
	if err != nil {
		x.LogErr(log, err).Fatal("While querying db")
		return rdoc
	}
	rdoc.Data = result.ToMap()
	return
}

func newUser() string {
	return "uid_" + x.UniqueString(3)
}

const sep = "================================"

func printAndGetUser(uid string) (user User) {
	result, err := store.NewQuery(uid).UptoDepth(10).Run(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	js, err := result.ToJson()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n%s\n%s\n%s\n", sep, string(js), sep)
	if err := json.Unmarshal(js, &user); err != nil || len(user.Post) == 0 {
		log.Fatalf("Error: %v", err)
	}
	return user
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Running...")
	flag.Parse()

	c = new(req.Context)
	c.NumCharsUnique = 10 // 62^10 permutations

	// Initialize leveldb.
	store.Get().Init("/tmp/ldb_" + x.UniqueString(10))
	// Initialize Elasticsearch.
	// search.Get().Init("http://192.168.59.103:9200")

	// Other possible initializations. Remember to import the right driver.
	// store.Get().Init("mysql", "root@tcp(127.0.0.1:3306)/test", "instructions")
	// store.Get().Init("192.168.59.103", "crudtest", "instructions")
	// store.Get().Init("192.168.59.103:27017", "crudtest", "instructions")
	// store.Get().Init("192.168.59.103:28015", "test", "instructions")

	search.Get().Init("memsearch")
	c.Indexer = SimpleIndexer{}
	c.RunIndexer(2)
	defer c.WaitForIndexer()

	log.Debug("Store initialized. Checking search...")
	uid := newUser()
	var err error

	// Let's get started. User 'uid' creates a new Post.
	// This Post shares a url, adds some text and some tags.
	tags := [3]string{"search", "cat", "videos"}
	err = store.NewUpdate("User", uid).SetSource(uid).AddChild("Post").
		Set("url", "www.google.com").Set("body", "You can search for cat videos here").
		Set("tags", tags).Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Print("Stored Post")

	// Now let's add a comment and two likes to our new post.
	// One user would add a comment and one like. Another user would
	// just like the post.
	//
	// It's best to have the same 'source' for one set of operations.
	// In REST APIs, this is how things would always be. Each REST call
	// is from one user (and never two different users).
	// This way the creation of like "entity", and the properties
	// of that new like entity have the same source.
	//
	// So, here's Step 1: A new user would add a comment, and like the post.
	user := printAndGetUser(uid)
	post := user.Post[0]

	p := store.NewUpdate("Post", post.Id).SetSource(newUser())
	p.AddChild("Like").Set("thumb", 1)
	p.AddChild("Comment").Set("body",
		fmt.Sprintf("Comment %s on the post", x.UniqueString(2)))
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 2: Another user would now like the post.
	p = store.NewUpdate("Post", post.Id).SetSource(newUser())
	p.AddChild("Like").Set("thumb", 1)
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Print("Added 1 Comment and 2 Like on Post")

	user = printAndGetUser(uid)
	post = user.Post[0]
	if len(post.Comment) == 0 {
		log.Fatalf("No comment found: %+v", post)
	}
	comment := post.Comment[0]

	// Now another user likes and replies to the comment that was added above.
	// So, it's a comment within a comment.
	p = store.NewUpdate("Comment", comment.Id).SetSource(newUser())
	p.AddChild("Like").Set("thumb", 1)
	p.AddChild("Comment").Set("body",
		fmt.Sprintf("Comment %s on comment", x.UniqueString(2)))
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Print("Added Comment on Comment")
	user = printAndGetUser(uid)
	post = user.Post[0]
	if len(post.Comment) == 0 {
		log.Fatalf("No comment found: %+v", post)
	}
	comment = post.Comment[0]
	if len(comment.Like) == 0 {
		log.Fatalf("No like found: %+v", comment)
	}
	like := comment.Like[0]

	// So far we have this structure:
	// User
	//  L Post
	//         L 2 * Like
	//         L Comment
	//            L Comment
	//            L Like

	// This is what most social platforms do. But, let's go
	// one level further, and also comment on the Likes on Comment.
	// User
	//    L Post
	//         L 2 * Like
	//         L Comment
	//            L Comment
	//            L Like
	//                 L Comment

	// Another user Comments on the Like on Comment on Post.

	p = store.NewUpdate("Like", like.Id).SetSource(newUser()).
		AddChild("Comment").Set("body",
		fmt.Sprintf("Comment %s on Like", x.UniqueString(2)))
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Print("Added Comment on Like")
	user = printAndGetUser(uid)

	{
		docs, err := search.Get().NewQuery("Like").Order("data.source").Run()
		if err != nil {
			x.LogErr(log, err).Fatal("While searching for Post")
			return
		}
		for _, doc := range docs {
			log.WithField("doc", doc).Debug("Resulting doc")
		}
		log.Debug("Search query over")
	}

	post = user.Post[0]
	if len(post.Comment) == 0 {
		log.Fatalf("No comment found: %+v", post)
	}
	comment = post.Comment[0]
	p = store.NewUpdate("Comment", comment.Id).SetSource(newUser()).Set("censored", true)
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	q := store.NewQuery(comment.Id).UptoDepth(0)
	result, err := q.Run(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	js, err := result.ToJson()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("\n%s\n%s\n%s\n", sep, string(js), sep)
	user = printAndGetUser(uid)

	post = user.Post[0]
	if len(post.Like) == 0 {
		log.Fatalf("No like found: %+v", post)
	}
	like = post.Like[0]
	p = store.NewUpdate("Like", like.Id).SetSource(newUser()).MarkDeleted()
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	q = store.NewQuery(uid).Collect("Post")
	q.Collect("Like").UptoDepth(10)
	q.Collect("Comment").UptoDepth(10).FilterOut("censored")
	result, err = q.Run(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	js, err = result.ToJson()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("\n%s\n%s\n%s\n", sep, string(js), sep)
	// By now we have a fairly complex Post structure. CRUD for
	// which would have been a lot of work to put together using
	// typical SQL / NoSQL tables.

	{
		docs, err := search.Get().NewQuery("Post").MatchExact("data.url", "www.google.com").Run()
		if err != nil {
			x.LogErr(log, err).Fatal("While searching for Post")
			return
		}
		for _, doc := range docs {
			log.WithField("doc", doc).Debug("Resulting doc")
		}
		log.Debug("Search query over")
	}
}
