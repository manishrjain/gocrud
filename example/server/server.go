package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/manishrjain/gocrud/api"
	"github.com/manishrjain/gocrud/helper"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("server")
var c *req.Context

func read(w http.ResponseWriter, r *http.Request) {
	id, ok := x.ParseIdFromUrl(r, "/read/")
	if !ok {
		return
	}

	// API usage to read data.
	q := api.NewQuery("hack", id).UptoDepth(10)
	result, err := q.Run(c)
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	result.WriteJsonResponse(w)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Running...")

	c = new(req.Context)
	l := new(store.Leveldb)
	l.SetBloomFilter(13)
	c.Store = l
	c.Store.Init("/tmp/crud_example_server")

	help := new(helper.Helper)
	help.SetContext(c)

	http.HandleFunc("/modify", help.CreateOrUpdate)
	http.HandleFunc("/read/", read)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		x.LogErr(log, err).Fatal("Creating listener")
	}
}
