package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	_ "gopkg.in/manishrjain/gocrud.v1/drivers/elasticsearch"
	"gopkg.in/manishrjain/gocrud.v1/search"
	"gopkg.in/manishrjain/gocrud.v1/x"
)

var eip = flag.String("ipaddr", "", "IP address of Elastic Search")
var num = flag.Int("num", 1, "Number of results")

type Author struct {
	Id string
	Ts int
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if *eip == "" {
		flag.Usage()
		return
	}

	engine := search.Get()
	engine.Init("http://" + *eip + ":9200")

	r := rand.Intn(100)
	uid := fmt.Sprintf("uid_%d", r)
	var au Author
	au.Id = fmt.Sprintf("mrjn-%d", r)
	au.Ts = r
	doc := x.Doc{Kind: "test", Id: uid, NanoTs: time.Now().UnixNano(), Data: au}
	if err := engine.Update(doc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	q := engine.NewQuery("test").Order("-Data.Ts").Limit(*num)
	q.NewAndFilter().AddExact("Data.Id", "mrjn")
	docs, err := q.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	for _, doc := range docs {
		fmt.Printf("Doc: %+v\n", doc)
	}
	return
}
