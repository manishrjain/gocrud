package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocql/gocql"
	_ "github.com/lib/pq"
	"github.com/manishrjain/gocrud/api"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
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

func newUser() string {
	return "uid_" + x.UniqueString(3)
}

const sep = "================================"

func printAndGetUser(uid string) (user User) {
	result, err := api.NewQuery("User", uid).UptoDepth(10).Run(c)
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

	if *storeType == "leveldb" {
		l := new(store.Leveldb)
		l.SetBloomFilter(13)
		c.Store = l
		c.Store.Init(*storeType, "/tmp/ldb_"+x.UniqueString(10))

	} else if *storeType == "cass" {
		cluster := gocql.NewCluster("192.168.59.103")
		cluster.Keyspace = "crudtest"
		cluster.Consistency = gocql.Quorum
		cass := new(store.Cassandra)

		if session, err := cluster.CreateSession(); err != nil {
			panic(err)
		} else {
			cass.SetSession(session)
		}
		c.Store = cass
		c.Store.Init(*storeType, "instructions")

	} else if *storeType == "mysql" {
		db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/test")
		if err != nil {
			panic(err)
		}
		if err = db.Ping(); err != nil {
			panic(err)
		}
		log.Info("Connection to mysql successful")
		sqldb := new(store.Sql)
		sqldb.SetDb(db)
		c.Store = sqldb
		c.Store.Init(*storeType, "instructions")

	} else if *storeType == "postgres" {
		db, err := sql.Open("postgres", "postgres://localhost/test?sslmode=disable")
		if err != nil {
			panic(err)
		}
		if err = db.Ping(); err != nil {
			panic(err)
		}
		log.Info("Connection to postgres successful")
		sqldb := new(store.Sql)
		sqldb.SetDb(db)
		c.Store = sqldb
		c.Store.Init(*storeType, "instructions")

	} else if *storeType == "datastore" {
		c.TablePrefix = "Test-"
		c.Store = new(store.Datastore)
		c.Store.Init(*storeType, "gce-project-id")

	} else {
		panic("Invalid store")
	}

	var err error
	uid := newUser()

	// Let's get started. User 'uid' creates a new Post.
	// This Post shares a url, adds some text and some tags.
	tags := [3]string{"search", "cat", "videos"}
	err = api.Get("User", uid).SetSource(uid).AddChild("Post").
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

	p := api.Get("Post", post.Id).SetSource(newUser())
	p.AddChild("Like").Set("thumb", 1)
	p.AddChild("Comment").Set("body",
		fmt.Sprintf("Comment %s on the post", x.UniqueString(2)))
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 2: Another user would now like the post.
	p = api.Get("Post", post.Id).SetSource(newUser())
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
	p = api.Get("Comment", comment.Id).SetSource(newUser())
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

	p = api.Get("Like", like.Id).SetSource(newUser()).
		AddChild("Comment").Set("body",
		fmt.Sprintf("Comment %s on Like", x.UniqueString(2)))
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Print("Added Comment on Like")
	user = printAndGetUser(uid)

	post = user.Post[0]
	if len(post.Comment) == 0 {
		log.Fatalf("No comment found: %+v", post)
	}
	comment = post.Comment[0]
	p = api.Get("Comment", comment.Id).SetSource(newUser()).Set("censored", true)
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	q := api.NewQuery("Comment", comment.Id).UptoDepth(0)
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
	p = api.Get("Like", like.Id).SetSource(newUser()).MarkDeleted()
	err = p.Execute(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	q = api.NewQuery("User", uid).Collect("Post")
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
}
