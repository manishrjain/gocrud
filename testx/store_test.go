package testx

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	_ "gopkg.in/manishrjain/gocrud.v1/drivers/leveldb"
	"gopkg.in/manishrjain/gocrud.v1/req"
	"gopkg.in/manishrjain/gocrud.v1/store"
)

func TestVersions(t *testing.T) {
	path, err := ioutil.TempDir("", "gocrudldb_")
	if err != nil {
		t.Fatal("Opening leveldb file")
		return
	}
	store.Get().Init(path) // leveldb

	c := req.NewContext(10)
	var d int
	for d = 660; d < 670; d++ {
		if err = store.NewUpdate("Ticker", "GOOG").SetSource("nasdaq").
			Set("price", d).Execute(c); err != nil {
			t.Errorf("When updating store: %+v", err)
			t.Fail()
		}
	}
	result, err := store.NewQuery("GOOG").Run()
	if err != nil {
		t.Errorf("When querying store: %+v", err)
		t.Fail()
	}
	versions, present := result.Columns["price"]
	if !present {
		t.Errorf("Column price should be present: %+v", result)
		t.Fail()
	}
	if versions.Count() != 10 {
		t.Errorf("Num count expected: 10. Got: %v", versions.Count())
	}
	if versions.Latest().Value.(float64) != 669 {
		t.Errorf("Latest value expected 669. Got: %v", versions.Latest())
	}
	if versions.Oldest().Value.(float64) != 660 {
		t.Errorf("Oldest value expected 660. Got: %v", versions.Oldest())
	}

	type jval struct {
		Kind   string `json:"kind,omitempty"`
		Id     string `json:"id,omitempty"`
		Source string `json:"source,omitempty"`
		Price  int    `json:"price,omitempty"`
	}
	js, err := result.ToJson()
	if err != nil {
		t.Errorf("While converting to JSON: %v", err)
	}
	t.Log(string(js))
	var jv jval
	if err = json.Unmarshal(js, &jv); err != nil {
		t.Errorf("While unmarshal to struct: %v", err)
	}
	if jv.Price != 669 {
		t.Errorf("Latest value expected 669. Got: %+v", jv)
	}
	if jv.Source != "nasdaq" {
		t.Errorf("Source expected nasdaq. Got: %+v", jv)
	}
}
