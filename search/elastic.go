package search

import (
	"errors"
	"reflect"

	"github.com/manishrjain/gocrud/x"
	"gopkg.in/olivere/elastic.v2"
)

type Elastic struct {
	client *elastic.Client
}

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

func (eq *ElasticQuery) Order(field string) SearchQuery {
	if field[:1] == "-" {
		eq.ss = eq.ss.Sort(field[1:], false)
	} else {
		eq.ss = eq.ss.Sort(field, true)
	}
	return eq
}

func (eq *ElasticQuery) Limit(num int) SearchQuery {
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

func (es *Elastic) NewQuery(kind string) SearchQuery {
	eq := new(ElasticQuery)
	eq.ss = es.client.Search("gocrud")
	return eq
}
