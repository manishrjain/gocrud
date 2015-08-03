package search

import (
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
	return nil
}

func (es *Elastic) NewQuery(kind string) *SearchQuery {
	return nil
}

func (es *Elastic) Run(query *SearchQuery) (docs []x.Doc, rerr error) {
	return
}
