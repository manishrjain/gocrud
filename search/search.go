package search

import "github.com/gocrud/x"

var log = x.Log("search")

type Entity struct {
	Kind string
	Id   string
}

type Doc struct {
	Kind   string
	Id     string
	Values map[string]interface{}
	NanoTs int64
}

// All the search operations are run via this Search interface.
// Implement this interface to add support for a search engine.
// Note that the term Entity is being used interchangeably with
// the term Subject. An Entity has a kind, and has an id.
type Updater interface {
	// OnUpdate is called when an entity is updated due to a Commit
	// on either itself, or it's direct children. Note that each
	// child entity would also be called with OnUpdate. This function
	// should return the Entity Ids, which need regeneration.
	OnUpdate(kind, id string) []Entity

	// Regenerate would be called on entities which need to be reprocessed
	// due to a change. The workflow is:
	// store.Commit -> search.OnUpdate -> Regenerate
	Regenerate(kind, id string) Doc
}

type Search struct {
}

func NewSearch(kind string) *Search { return nil }

// Search docs where:
// Where("field =", "something") or
// Where("field >", "something") or
// Where("field <", "something")
func (s *Search) Where(field, value interface{}) {}
func (s *Search) Limit(num int)                  {}
func (s *Search) Order(field string)             {}
