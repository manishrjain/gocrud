package main

import (
	"gopkg.in/manishrjain/gocrud.v1/req"
	"gopkg.in/manishrjain/gocrud.v1/store"
)

// Create a new Meetup event.
func main() {
	// Get entity, set author for update.
	u := store.NewUpdate("Meetup", "Events").SetSource("ScottBarr")

	p := u.AddChild("Event") // Add Event to Meetup entity.
	p.Set("location", "River City Labs").Set("city", "Brisbane").
		Set("topic", "First Brisbane Gophers meetup") // Method chaining

	// Add attendees to Event entity.
	p.AddChild("Attendee").Set("name", "Scott").Set("organizer", true)
	p.AddChild("Attendee").Set("name", "Manish").Set("presenter", true)

	ctx := req.NewContext(3) // Number of chars in unique ids.
	_ = u.Execute(ctx)       // Returns error
}
