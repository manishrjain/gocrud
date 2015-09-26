package main

import (
	"gopkg.in/manishrjain/gocrud.v1/req"
	"gopkg.in/manishrjain/gocrud.v1/store"
)

// Create a new Social Post.
func update() {
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

	ctx := req.NewContext(3) // Number of chars in unique ids.
	_ = u.Execute(ctx)       // Returns error
}

func query() {
	// Collect all Post entities under User userid.
	p := store.NewQuery("userid").Collect("Post")
	p.Collect("Like") // Collect all Like entities for Post entities.

	// Collect all Comment entities, and Likes on those comments.
	p.Collect("Comment").Collect("Like")

	// Alternatively, get everything up to certain depth.
	p = store.NewQuery("userid").UptoDepth(10)

	result, _ := p.Run()
	result.ToJson() // Convert to JSON.

	// Optionally write to w http.ResponseWriter.
	result.WriteJsonResponse(w)
}

func main() {
}
