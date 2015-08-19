package memsearch

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/manishrjain/gocrud/search"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("memsearch")

type MemSearch struct {
	docs map[string]x.Doc
}

type MemQuery struct {
	kind string
	Docs []x.Doc
}

func (ms *MemSearch) Init(args ...string) {
	ms.docs = make(map[string]x.Doc)
}

func (ms *MemSearch) All() []x.Doc {
	var dup []x.Doc
	for _, doc := range ms.docs {
		dup = append(dup, doc)
	}
	return dup
}

func (ms *MemSearch) NewQuery(kind string) search.Query {
	mq := new(MemQuery)
	for _, doc := range ms.docs {
		if doc.Kind != kind {
			continue
		}
		mq.Docs = append(mq.Docs, doc)
	}
	return mq
}

func (ms *MemSearch) Update(doc x.Doc) error {
	key := doc.Kind + ":" + doc.Id
	if pdoc, present := ms.docs[key]; present {
		if pdoc.NanoTs >= doc.NanoTs {
			return errors.New("version conflict")
		}
	}
	ms.docs[key] = doc
	return nil
}

func (mq *MemQuery) MatchExact(field string, value interface{}) search.Query {
	if len(field) > len("data.") && strings.ToLower(field[0:5]) == "data." {
		field = field[5:]
	}
	filtered := mq.Docs[:0]

	for _, doc := range mq.Docs {
		fields := doc.Data.(map[string]interface{})
		if val, present := fields[field]; present {
			if match := reflect.DeepEqual(val, value); match {
				log.WithFields(logrus.Fields{
					"field": field,
					"doc":   doc.Id,
					"value": value,
				}).Debug("Matches")
				filtered = append(filtered, doc)
				continue
			}
		}
	}
	mq.Docs = filtered
	log.WithField("field", field).Debug("Done Matching")
	return mq
}

func (mq *MemQuery) Limit(num int) search.Query {
	if len(mq.Docs) > num {
		mq.Docs = mq.Docs[0:num]
	}
	return mq
}

type Docs struct {
	data  []x.Doc
	field string
	fn    func(i, j x.Doc)
}

func (d Docs) Len() int      { return len(d.data) }
func (d Docs) Swap(i, j int) { d.data[i], d.data[j] = d.data[j], d.data[i] }
func (d Docs) Get(i int) (val interface{}) {
	di := d.data[i]
	fi := di.Data.(map[string]interface{})
	vi, pi := fi[d.field]
	if !pi {
		log.WithFields(logrus.Fields{
			"field": d.field,
			"data":  fi,
		}).Fatal("Field not found for sorting")
		return nil
	}
	return vi
}
func (d Docs) Less(i, j int) bool {
	vi := d.Get(i)
	vj := d.Get(j)
	if reflect.TypeOf(vi) != reflect.TypeOf(vj) {
		log.WithFields(logrus.Fields{
			"vi":    vi,
			"vj":    vj,
			"field": d.field,
		}).Fatal("Different types")
		return false
	}
	switch t := vi.(type) {
	case string:
		return vi.(string) < vj.(string)
	case int64, int32, int:
		return vi.(int64) < vj.(int64)
	case float64:
		return vi.(float64) < vj.(float64)
	default:
		log.WithFields(logrus.Fields{
			"vi":         vi,
			"vj":         vj,
			"field":      d.field,
			"type_found": fmt.Sprintf("%T", t),
		}).Fatal("Invalid type")
	}

	return false
}

func (mq *MemQuery) Order(field string) search.Query {
	reverse := false
	if strings.HasPrefix(field, "-") {
		reverse = true
		field = field[1:]
	}
	if len(field) > len("data.") && strings.ToLower(field[0:5]) == "data." {
		field = field[5:]
	}

	eligible := mq.Docs[:0]
	for _, doc := range mq.Docs {
		fi := doc.Data.(map[string]interface{})
		if _, pi := fi[field]; pi {
			eligible = append(eligible, doc)
		}
	}
	mq.Docs = eligible

	docs := Docs{data: mq.Docs, field: field}
	if reverse {
		sort.Sort(sort.Reverse(docs))
	} else {
		sort.Sort(docs)
	}
	return mq
}

func (mq *MemQuery) Run() (docs []x.Doc, rerr error) {
	return mq.Docs, nil
}

func init() {
	log.Info("Initing memsearch")
	search.Register("memsearch", new(MemSearch))
}
