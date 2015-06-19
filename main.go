package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/crud/api"
	"github.com/crud/req"
	"github.com/crud/store"
	"github.com/crud/x"
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
		api.Get("Review", "rid").
			AddChild("Comment").SetSource("manish").
			SetText("value", "this is a comment").
			SetText("active", "true").Execute(&c)
	*/
	api.Get("Comment", "w56fk").SetSource("manish").
		SetText("censored", "false").Execute(&c)

	handleFunc("/commit", api.Handle, &c)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		x.LogErr(log, err).Fatal("Creating listener")
	}
}
