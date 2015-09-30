package elasticsearch

import (
	"errors"
	"reflect"

	"gopkg.in/manishrjain/gocrud.v1/search"
	"gopkg.in/manishrjain/gocrud.v1/x"
	"gopkg.in/olivere/elastic.v2"
)

var log = x.Log("elasticsearch")

// Elastic encapsulates elastic search client, and implements methods declared
// by search.Engine.
type Elastic struct {
	client *elastic.Client
}

// ElasticQuery implements methods declared by search.Query.
type ElasticQuery struct {
	client     *elastic.Client
	sort       string
	limit      int
	kind       string
	filter     *ElasticFilter
	filterType int // 0 = no filter, 1 = AND, 2 = OR
}

type ElasticFilter struct {
	filters []elastic.Filter
}

// Init initializes connection to Elastic Search instance, checks for
// existence of "gocrud" index and creates it, if missing. Note that
// Init does NOT do mapping necessary to do exact-value term matching
// for strings etc. That needs to be done externally.
func (es *Elastic) Init(args ...string) {
	if len(args) != 1 {
		log.WithField("args", args).Fatal("Invalid arguments")
		return
	}
	url := args[0]

	log.Debug("Initializing connection to ElaticSearch")
	var opts []elastic.ClientOptionFunc
	opts = append(opts, elastic.SetURL(url))
	opts = append(opts, elastic.SetSniff(false))
	client, err := elastic.NewClient(opts...)
	if err != nil {
		x.LogErr(log, err).Fatal("While creating connection with ElaticSearch.")
		return
	}
	version, err := client.ElasticsearchVersion(url)
	if err != nil {
		x.LogErr(log, err).Fatal("Unable to query version")
		return
	}
	log.WithField("version", version).Debug("ElasticSearch version")

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists("gocrud").Do()
	if err != nil {
		x.LogErr(log, err).Fatal("Unable to query index existence.")
		return
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("gocrud").Do()
		if err != nil {
			x.LogErr(log, err).Fatal("Unable to create index.")
			return
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
			log.Errorf("Create index not acknowledged. Not sure what that means...")
		}
	}
	es.client = client
	log.Debug("Connected with ElasticSearch")
}

// DropIndex is useful for testing purposes.
func (es *Elastic) DropIndex() error {
	_, err := es.client.DeleteIndex("gocrud").Do()
	return err
}

// Update checks the validify of given document, and the.
// external versioning via the timestamp of the document.
func (es *Elastic) Update(doc x.Doc) error {
	if doc.Id == "" || doc.Kind == "" || doc.NanoTs == 0 {
		return errors.New("Invalid document")
	}

	result, err := es.client.Index().Index("gocrud").Type(doc.Kind).Id(doc.Id).
		VersionType("external").Version(doc.NanoTs).BodyJson(doc).Do()
	if err != nil {
		x.LogErr(log, err).WithField("doc", doc).Error("While indexing doc")
		return err
	}
	log.Debug("index_result", result)
	return nil
}

func (eq *ElasticQuery) NewAndFilter() search.FilterQuery {
	eq.filter = new(ElasticFilter)
	eq.filterType = 1

	return eq.filter
}

func (eq *ElasticQuery) NewOrFilter() search.FilterQuery {
	eq.filter = new(ElasticFilter)
	eq.filterType = 2

	return eq.filter
}

// AddExact implemented by ElasticSearch uses the 'term' directive.
// Note that with strings, this might not return exact match results,
// if the index is set to pre-process strings, which it does by default.
// In other words, for string term-exact matches to work, you need to
// set the mapping to "index": "not_analyzed".
// https://www.elastic.co/guide/en/elasticsearch/guide/current/mapping-intro.html
func (ef *ElasticFilter) AddExact(field string,
	value interface{}) search.FilterQuery {

	tq := elastic.NewTermFilter(field, value)
	ef.filters = append(ef.filters, tq)
	return ef
}

// AddRegex uses regex filter directive.
func (ef *ElasticFilter) AddRegex(field string,
	value string) search.FilterQuery {

	wq := elastic.NewRegexpFilter(field, value)
	ef.filters = append(ef.filters, wq)
	return ef
}

// Order sorts the results for the given field.
func (eq *ElasticQuery) Order(field string) search.Query {
	eq.sort = field
	return eq
}

// Limit limits the number of results to num.
func (eq *ElasticQuery) Limit(num int) search.Query {
	eq.limit = num
	return eq
}

// Run runs the query and returns results and error, if any.
func (eq *ElasticQuery) Run() (docs []x.Doc, rerr error) {
	ss := eq.client.Search("gocrud").Type(eq.kind)
	if len(eq.sort) > 0 {
		if eq.sort[:1] == "-" {
			ss = ss.Sort(eq.sort[1:], false)
		} else {
			ss = ss.Sort(eq.sort, true)
		}
	}
	if eq.limit > 0 {
		ss = ss.Size(eq.limit)
	}

	if eq.filter != nil {
		q := elastic.NewFilteredQuery(elastic.NewMatchAllQuery())
		if eq.filterType == 0 {
			return docs, errors.New("Filter present, but not set")

		} else if eq.filterType == 1 {
			af := elastic.NewAndFilter(eq.filter.filters...)
			q = q.Filter(af)

		} else if eq.filterType == 2 {
			of := elastic.NewOrFilter(eq.filter.filters...)
			q = q.Filter(of)

		} else {
			return docs, errors.New("Invalid filter type")
		}

		ss = ss.Query(q)
	}

	result, err := ss.Do()
	if err != nil {
		x.LogErr(log, err).Error("While running query")
		return docs, err
	}
	if result.Hits == nil {
		log.Debug("No results found")
		return docs, nil
	}

	var d x.Doc
	for _, item := range result.Each(reflect.TypeOf(d)) {
		d := item.(x.Doc)
		docs = append(docs, d)
	}
	return docs, nil
}

// NewQuery creates a new query object, to return results of type kind.
func (es *Elastic) NewQuery(kind string) search.Query {
	eq := new(ElasticQuery)
	eq.client = es.client
	eq.kind = kind
	return eq
}

func init() {
	log.Info("Initing elasticsearch")
	search.Register("elasticsearch", new(Elastic))
}
