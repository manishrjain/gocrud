package testx

import (
	"log"
	"testing"
	"time"

	"gopkg.in/manishrjain/gocrud.v1/search"
	"gopkg.in/manishrjain/gocrud.v1/x"
)

var galaxies = [...]string{
	"sombrero galaxy", "messier 64", "2masx",
	"whirlpool galaxy", "ngc 123", "supernova",
	"galaxy ngc 1512", "ngc 3370", "m81",
}

func AddDocs(e search.Engine) {
	for idx, name := range galaxies {
		var d x.Doc
		d.Id = x.UniqueString(5)
		d.Kind = "Galaxy"
		d.NanoTs = time.Now().UnixNano()
		m := make(map[string]interface{})
		m["name"] = name
		m["pos"] = idx
		d.Data = m

		if err := e.Update(d); err != nil {
			log.Fatalf("While updating: %v", err)
			return
		}
	}
}

func RunAndFilter(e search.Engine, t *testing.T) {
	q := e.NewQuery("Galaxy")
	q.NewAndFilter().AddExact("name", "2masx").AddRegex("name", ".*ma.*")
	docs, err := q.Run()
	if err != nil {
		t.Fatalf("While running query: %v", err)
		return
	}
	if len(docs) != 1 {
		t.Errorf("Number of docs should be 1. Found: %v\n", len(docs))
	} else {
		d := docs[0]
		m := d.Data.(map[string]interface{})
		val, found := m["name"]
		if !found {
			t.Error("Should find name")
		} else {
			if val.(string) != "2masx" {
				t.Errorf("Expected 2masx. Found: %v\n", val.(string))
			}
		}
	}
}

var soln = [...]string{
	"m81",
	"ngc 3370",
	"galaxy ngc 1512",
	"ngc 123",
	"whirlpool galaxy",
	"sombrero galaxy",
}

func RunOrFilter(e search.Engine, t *testing.T) {
	q := e.NewQuery("Galaxy").Order("-pos")
	q.NewOrFilter().AddRegex("name", ".*galaxy.*").
		AddRegex("name", ".*ngc.*").AddExact("name", "m81")
	docs, err := q.Run()
	if err != nil {
		t.Fatalf("While running query: %v", err)
		return
	}
	if len(docs) != 6 {
		t.Errorf("Number of docs should be %v. Found: %v\n", len(soln), len(docs))
	} else {
		for idx, doc := range docs {
			m := doc.Data.(map[string]interface{})
			val, found := m["name"]
			if !found {
				t.Error("Should find name")
			} else {
				if val.(string) != soln[idx] {
					t.Errorf("Expected: %v. Found: %v\n", soln[idx], val.(string))
				}
			}
		}
	}
}
