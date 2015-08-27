// Package search provides a way to index entities and run relatively
// complicated search queries, best served outside of data stores, and
// by specialized search engines like ElasticSearch or Solr etc.
//
// This tackles the limitations caused by gocrud in terms of filtering
// and sort operations which would otherwise would need to be done at
// application level.
package search

import "github.com/manishrjain/gocrud/x"

var log = x.Log("search")

// All the search operations are run via this Search interface.
// Implement this interface to add support for a search engine.
// Note that the term Entity is being used interchangeably with
// the term Subject. An Entity has a kind, and has an id.

// Query interface provides the search api encapsulator, responsible for
// generating the right query for the engine, and then running it.
type Query interface {
	// MatchExact would do exact full string, int, etc. matching. Also called
	// term matching by some engines.
	MatchExact(field string, value interface{}) Query

	// MatchPartial would do wild card matching. Requires value to be string,
	// typically in the format: [sear*], or [*arch], or [*arc*].
	MatchPartial(field string, value string) Query

	// Limit would limit the number of results to num.
	Limit(num int) Query

	// Order would sort the results by field in ascending order.
	// A "-field" can be provided to sort results in descending order.
	Order(field string) Query

	// Run the generated query, providing resulting documents and error, if any.
	Run() ([]x.Doc, error)
}

// Engine provides the interface to be implemented to support search engines.
type Engine interface {
	// Init should be used for initializing search engine. The string arguments
	// can be used differently by different engines.
	Init(args ...string)

	// Update doc into index. Note that doc.NanoTs should be utilized to implement
	// any sort of versioning facility provided by the search engine, to avoid
	// overwriting a newer doc by an older doc.
	Update(x.Doc) error

	// NewQuery creates the query encapsulator, restricting results by given kind.
	NewQuery(kind string) Query
}

// Search docs where:
// Where("field =", "something") or
// Where("field >", "something") or
// Where("field <", "something")

var dengine Engine

func Register(name string, driver Engine) {
	if driver == nil {
		log.WithField("search", name).Fatal("nil engine")
		return
	}
	if dengine != nil {
		log.WithField("search", name).Fatal("Register called twice")
		return
	}

	log.WithField("search", name).Debug("Registering search engine")
	dengine = driver
}

func Get() Engine {
	if dengine == nil {
		log.Fatal("No engine registered")
		return nil
	}
	return dengine
}
