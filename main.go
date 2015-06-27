package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gocrud/api"
	"github.com/gocrud/req"
	"github.com/gocrud/store"
	"github.com/gocrud/x"
)

var log = x.Log("main")

type HandlerFunc func(http.ResponseWriter, *http.Request, *req.Context)

func handleFunc(name string, fn HandlerFunc, c *req.Context) {
	http.HandleFunc(name, addDefaultHeaders(fn, c))
}

func addDefaultHeaders(fn HandlerFunc, c *req.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token,"+
				" X-Auth-Token, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method != "OPTIONS" {
			w.Header().Set("Content-Type", "application/json")
			fn(w, r, c)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Running...")

	c := req.Context{}
	c.TablePrefix = "MRJN-"
	c.Store = new(store.Datastore)
	c.Store.Init("supportx-backend")

	/*
		if err := api.Get("Review", "rid").
			SetSource("mrjn").
			Set("active", true).
			Set("text", "this is my review").
			Execute(&c); err != nil {
			log.Errorf("While creating review: %v", err)
		}
	*/

	/*
		n := api.Get("Review", "rid").SetSource("andi")
		n.AddChild("Comment").Set("text", "this is my comment")
		n.AddChild("Vote").Set("value", 1)
		log.Infof("Err while executing: %v", n.Execute(&c))
	*/

	// api.Get("Review", "rid").JsonGraph(&c)

	/*
		api.Get("Comment", "dgu6d").
			SetSource("mrjn").AddChild("Vote").
			Set("value", rand.Intn(10)).Execute(&c)
	*/

	api.Get("Vote", "ltkdw").SetSource("andi").Set("inactive", true).Execute(&c)
	q := api.NewQuery("Review", "rid")
	q.Collect("Comment")
	q.Collect("Vote").FilterOut("inactive")
	result, err := q.Run(&c)
	if err != nil {
		log.Errorf("While querying: %v", err)
	} else {
		data, err := result.ToJson()
		log.Infof("Err: %v. Data: %+v", err, string(data))
	}

	/*
		api.Get("Review", "rid").
			AddChild("Comment").SetSource("manish").
			SetText("value", "this is a comment").
			SetText("active", "true").Execute(&c)
	*/
	/*
		api.Get("Comment", "w56fk").SetSource("manish").
			SetText("censored", "false").Execute(&c)
	*/
	/*
		api.Get("Comment", "w56fk").JsonGraph(&c)
	*/

	if err := http.ListenAndServe(":8080", nil); err != nil {
		x.LogErr(log, err).Fatal("Creating listener")
	}
}
