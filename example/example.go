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

type Post struct {
	Url  string   `json:"url,omitempty"`
	Body string   `json:"body,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type Comment struct {
	PostId    string `json:"post_id,omitempty"`
	CommentId string `json:"comment_id,omitempty"`
	Body      string `json:"body,omitempty"`
}

type Like struct {
	CommentId string `json:"comment_id,omitempty"`
	PostId    string `json:"post_id,omitempty"`
	Thumb     bool   `json:"thumb,omitempty"`
}

func auth(r *http.Request) string {
	return "user_id"
}

func newPost(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var post Post
	if ok := x.ParseRequest(w, r, &post); !ok {
		return
	}

	p := api.Get("User", uid).SetSource(uid).AddChild("Post")
	p.Set("url", post.Url).Set("body", post.Body).Set("tags", post.Tags)
	if err := p.Execute(c); err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}

	x.SetStatus(w, x.E_OK, "Stored")
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	userid, ok := x.ParseIdFromUrl(r, "/posts/")
	if !ok {
		x.SetStatus(w, x.E_INVALID_REQUEST, "Unable to find post id")
		return
	}
	// TODO: Fix this. Doesn't show the comment.
	result, err := api.NewQuery("User", userid).Collect("Post").Collect("Comment").Collect("Comment").Run(c)
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	result.WriteJsonResponse(w)
}

func newComment(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var comment Comment
	if ok := x.ParseRequest(w, r, &comment); !ok {
		return
	}
	var err error
	if len(comment.PostId) > 0 {
		err = api.Get("Post", comment.PostId).SetSource(uid).
			AddChild("Comment").Set("body", comment.Body).Execute(c)
	} else if len(comment.CommentId) > 0 {
		err = api.Get("Comment", comment.CommentId).SetSource(uid).
			AddChild("Comment").Set("body", comment.Body).Execute(c)
	}
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	x.SetStatus(w, x.E_OK, "Stored")
}

func newLike(w http.ResponseWriter, r *http.Request) {
	uid := auth(r)
	var like Like
	if ok := x.ParseRequest(w, r, &like); !ok {
		return
	}
	var err error
	if len(like.CommentId) > 0 {
		err = api.Get("Comment", like.CommentId).SetSource(uid).
			AddChild("Like").Set("thumb", like.Thumb).Execute(c)
	} else if len(like.PostId) > 0 {
		err = api.Get("Post", like.PostId).SetSource(uid).
			AddChild("Like").Set("thumb", like.Thumb).Execute(c)
	}
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	x.SetStatus(w, x.E_OK, "Stored")
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Running...")

	c = new(req.Context)
	c.TablePrefix = "MRJN-"
	c.Store = new(store.Datastore)
	c.Store.Init("supportx-backend")

	http.HandleFunc("/posts", newPost)
	http.HandleFunc("/comments", newComment)
	http.HandleFunc("/likes", newLike)
	http.HandleFunc("/posts/", getPosts)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		x.LogErr(log, err).Fatal("Creating listener")
	}
}
