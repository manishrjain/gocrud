package elasticsearch

import (
	"os"
	"testing"

	"github.com/manishrjain/gocrud/testx"
)

var galaxies = [...]string{
	"sombrero galaxy", "messier 64", "2masx",
	"whirlpool galaxy", "ngc 123", "supernova",
	"galaxy ngc 1512", "ngc 3370", "m81",
}

func initialize(t *testing.T) *Elastic {
	addr := os.Getenv("ELASTICSEARCH_PORT_9200_TCP_ADDR")
	if len(addr) == 0 {
		t.Log("Elastic Search environment vars not set")
		return nil
	}

	es := new(Elastic)
	es.Init("http://" + addr)
	testx.AddDocs(es, t)
	return es
}

func TestNewAndQuery(t *testing.T) {
	es := initialize(t)
	if es == nil {
		return
	}
	testx.RunAndFilter(es, t)
}

func TestNewOrFilter(t *testing.T) {
	es := initialize(t)
	if es == nil {
		return
	}
	testx.RunOrFilter(es, t)
}
