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

var log = x.Log("example")
var c *req.Context

type Create struct {
	Kind       string                 `json:"kind,omitempty"`
	ParentId   string                 `json:"parent_id,omitempty"`
	ParentKind string                 `json:"parent_kind,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

type Update struct {
	Kind string                 `json:"kind,omitempty"`
	Id   string                 `json:"id,omitempty"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type Read struct {
	Kind     string `json:"kind,omitempty"`
	Id       string `json:"id,omitempty"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

func auth(r *http.Request) string {
	return "user_id"
}

func create(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var req Create
	if ok := x.ParseRequest(w, r, &req); !ok {
		return
	}

	// API usage to create data.
	p := api.Get(req.ParentKind, req.ParentId).SetSource(uid).AddChild(req.Kind)
	for key, val := range req.Data {
		p.Set(key, val)
	}
	if err := p.Execute(c); err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	x.SetStatus(w, x.E_OK, "Stored")
}

func update(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var up Update
	if ok := x.ParseRequest(w, r, &up); !ok {
		return
	}

	// API usage to update data.
	p := api.Get(up.Kind, up.Id).SetSource(uid)
	for key, val := range up.Data {
		p.Set(key, val)
	}
	if err := p.Execute(c); err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	x.SetStatus(w, x.E_OK, "Stored")
}

func read(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var read Read
	if ok := x.ParseRequest(w, r, &read); !ok {
		return
	}

	// API usage to read data.
	q := api.NewQuery(read.Kind, read.Id).UptoDepth(read.MaxDepth)
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
	c.TablePrefix = "MRJN-"
	c.Store = new(store.Datastore)
	c.Store.Init("supportx-backend")

	http.HandleFunc("/create", create)
	http.HandleFunc("/update", update)
	http.HandleFunc("/read", read)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		x.LogErr(log, err).Fatal("Creating listener")
	}
}
