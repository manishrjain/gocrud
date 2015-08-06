package search

import (
	"errors"
	"reflect"

	"github.com/manishrjain/gocrud/x"
	"gopkg.in/olivere/elastic.v2"
)

// Elastic encapsulates elastic search client, and implements methods declared
// by search.Engine.
type Elastic struct {
	client *elastic.Client
}

// Init initializes connection to Elastic Search instance, checks for
// existence of "gocrud" index and creates it, if missing. Note that
// Init does NOT do mapping necessary to do exact-value term matching
// for strings etc. That needs to be done externally.
func (es *Elastic) Init(url string) {
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

type ElasticQuery struct {
	ss *elastic.SearchService
}

// MatchExact implemented by ElasticSearch uses the 'term' directive.
// Note that with strings, this might not return exact match results,
// if the index is set to pre-process strings, which it does by default.
// In other words, for string term-exact matches to work, you need to
// set the mapping to "index": "not_analyzed".
// https://www.elastic.co/guide/en/elasticsearch/guide/current/mapping-intro.html
func (eq *ElasticQuery) MatchExact(field string,
	value interface{}) Query {
	tq := elastic.NewTermQuery(field, value)
	eq.ss = eq.ss.Query(&tq)
	return eq
}

func (eq *ElasticQuery) Order(field string) Query {
	if field[:1] == "-" {
		eq.ss = eq.ss.Sort(field[1:], false)
	} else {
		eq.ss = eq.ss.Sort(field, true)
	}
	return eq
}

func (eq *ElasticQuery) Limit(num int) Query {
	eq.ss = eq.ss.Size(num)
	return eq
}

func (eq *ElasticQuery) Run() (docs []x.Doc, rerr error) {
	result, err := eq.ss.Do()
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

func (es *Elastic) NewQuery(kind string) Query {
	eq := new(ElasticQuery)
	eq.ss = es.client.Search("gocrud").Type(kind)
	return eq
}
