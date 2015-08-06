package search

import "github.com/manishrjain/gocrud/x"

var log = x.Log("search")

// All the search operations are run via this Search interface.
// Implement this interface to add support for a search engine.
// Note that the term Entity is being used interchangeably with
// the term Subject. An Entity has a kind, and has an id.
type Updater interface {
	// OnUpdate is called when an entity is updated due to a Commit
	// on either itself, or it's direct children. Note that each
	// child entity would also be called with OnUpdate. This function
	// should return the Entity Ids, which need regeneration.
	OnUpdate(kind, id string) []x.Entity

	// Regenerate would be called on entities which need to be reprocessed
	// due to a change. The workflow is:
	// store.Commit -> search.OnUpdate -> Regenerate
	Regenerate(kind, id string) x.Doc
}

type SearchQuery interface {
	MatchExact(field string, value interface{}) SearchQuery
	Limit(num int) SearchQuery
	Order(field string) SearchQuery
	Run() ([]x.Doc, error)
}

type Engine interface {
	Init(string)
	Update(x.Doc) error
	NewQuery(kind string) SearchQuery
}

// Search docs where:
// Where("field =", "something") or
// Where("field >", "something") or
// Where("field <", "something")
