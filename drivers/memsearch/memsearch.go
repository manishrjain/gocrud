package memsearch

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
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
	kind       string
	Docs       []x.Doc
	filter     *MemFilter
	filterType int // 0 = no filter, 1 = AND, 2 = OR
	limit      int
	order      string
}

type Filter struct {
	Field string
	Value interface{}
	Regex string
}

type MemFilter struct {
	filters []Filter
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

func (mq *MemQuery) NewAndFilter() search.FilterQuery {
	mq.filter = new(MemFilter)
	mq.filterType = 1 // AND
	return mq.filter
}

func (mq *MemQuery) NewOrFilter() search.FilterQuery {
	mq.filter = new(MemFilter)
	mq.filterType = 2 // OR
	return mq.filter
}

func (mf *MemFilter) AddExact(field string,
	value interface{}) search.FilterQuery {

	filter := Filter{Field: field, Value: value}
	mf.filters = append(mf.filters, filter)
	return mf
}

func (mf *MemFilter) AddRegex(field string,
	value string) search.FilterQuery {

	filter := Filter{Field: field, Regex: value}
	mf.filters = append(mf.filters, filter)
	return mf
}

func matchExact(doc x.Doc, field string, value interface{}) bool {
	if len(field) > len("data.") && strings.ToLower(field[0:5]) == "data." {
		field = field[5:]
	}

	fields := doc.Data.(map[string]interface{})
	if val, present := fields[field]; present {
		if match := reflect.DeepEqual(val, value); match {
			return true
		}
	}
	return false
}

func matchRegex(doc x.Doc, field, value string) bool {
	re := regexp.MustCompile(value)
	if len(field) > len("data.") && strings.ToLower(field[0:5]) == "data." {
		field = field[5:]
	}

	fields := doc.Data.(map[string]interface{})
	if val, present := fields[field]; present {
		vals := val.(string)

		if re.MatchString(vals) {
			return true
		}
	}
	return false
}

func (mq *MemQuery) Limit(num int) search.Query {
	mq.limit = num
	return mq
}

func (mq *MemQuery) Order(field string) search.Query {
	mq.order = field
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
	case int64:
		return vi.(int64) < vj.(int64)
	case int32:
		return vi.(int32) < vj.(int32)
	case int:
		return vi.(int) < vj.(int)
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

func (mq *MemQuery) bringOrder(field string) search.Query {
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

func (mq *MemQuery) runAndFilter(filters []Filter) error {
	docs := mq.Docs
	filtered := docs[:0]
	for _, doc := range docs {

		for _, f := range filters {
			if len(f.Field) == 0 {
				return errors.New("Invalid field")
			}

			if len(f.Regex) > 0 {
				if m := matchRegex(doc, f.Field, f.Regex); m {
					filtered = append(filtered, doc)
					break // from filters.
				}
			} else {
				if m := matchExact(doc, f.Field, f.Value); m {
					filtered = append(filtered, doc)
					break // from filters.
				}
			}
		}
	}
	mq.Docs = filtered
	return nil
}

func (mq *MemQuery) runOrFilter(filters []Filter) error {
	docs := mq.Docs
	filtered := docs[:0]
	for _, doc := range docs {

		match := false
		for _, f := range filters {
			if len(f.Field) == 0 {
				return errors.New("Invalid field")
			}

			if len(f.Regex) > 0 {
				if matchRegex(doc, f.Field, f.Regex) {
					match = true
					break // from filters
				}
			} else {
				if matchExact(doc, f.Field, f.Value) {
					match = true
					break // from filters
				}
			}
		}

		if match {
			filtered = append(filtered, doc)
		}
	}
	mq.Docs = filtered
	return nil
}

func (mq *MemQuery) runFilter() error {
	if mq.filterType == 0 {
		return errors.New("Filter present, but not set")

	} else if mq.filterType == 1 {
		if err := mq.runAndFilter(mq.filter.filters); err != nil {
			return err
		}
	} else if mq.filterType == 2 {
		if err := mq.runOrFilter(mq.filter.filters); err != nil {
			return err
		}
	} else {
		return errors.New("Invalid filter type")
	}
	return nil
}

func (mq *MemQuery) Run() (docs []x.Doc, rerr error) {
	if mq.filter != nil {
		if err := mq.runFilter(); err != nil {
			return docs, err
		}
	}
	if len(mq.order) > 0 {
		mq.bringOrder(mq.order)
	}
	if mq.limit > 0 && len(mq.Docs) > mq.limit {
		mq.Docs = mq.Docs[0:mq.limit]
	}
	return mq.Docs, nil
}

func (mq *MemQuery) Count() (rcount int64, rerr error) {
	if mq.filter != nil {
		if err := mq.runFilter(); err != nil {
			return 0, err
		}
	}
	return int64(len(mq.Docs)), nil
}

func init() {
	log.Info("Initing memsearch")
	search.Register("memsearch", new(MemSearch))
}
