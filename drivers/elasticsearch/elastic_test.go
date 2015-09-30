package elasticsearch

import (
	"os"
	"testing"
	"time"

	"github.com/manishrjain/gocrud/testx"
)

var galaxies = [...]string{
	"sombrero galaxy", "messier 64", "2masx",
	"whirlpool galaxy", "ngc 123", "supernova",
	"galaxy ngc 1512", "ngc 3370", "m81",
}

func initialize() *Elastic {
	addr := os.Getenv("ELASTICSEARCH_PORT_9200_TCP_ADDR")
	if len(addr) == 0 {
		return nil
	}

	es := new(Elastic)
	es.Init("http://" + addr + ":9200")
	es.DropIndex()
	testx.AddDocs(es)
	return es
}

func TestNewAndQuery(t *testing.T) {
	if es == nil {
		t.Log("Elastic Search environment vars not set")
		return
	}
	testx.RunAndFilter(es, t)
}

func TestNewOrFilter(t *testing.T) {
	if es == nil {
		t.Log("Elastic Search environment vars not set")
		return
	}
	testx.RunOrFilter(es, t)
}

func TestCount(t *testing.T) {
	if es == nil {
		t.Log("Elastic Search environment vars not set")
		return
	}
	testx.RunCount(es, t)
}

func TestFrom(t *testing.T) {
	if es == nil {
		t.Log("Elastic Search environment vars not set")
		return
	}
	testx.RunFromLimit(es, t)
}

var es *Elastic

func init() {
	es = initialize()
	if es == nil {
		return
	}
	time.Sleep(5 * time.Second) // To allow updates to become available for search.
}
