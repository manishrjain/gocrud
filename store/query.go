package store

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/manishrjain/gocrud/x"
)

var (
	ErrNoParent = errors.New("No parent found")
)

// Query stores the read instrutions, storing the instruction set
// for the entities Query relates to.
type Query struct {
	kind      string
	id        string
	filterOut map[string]bool
	maxDepth  int
	children  []*Query
	parent    *Query
}

type Object struct {
	Value  interface{}
	Source string
	NanoTs int64
}

type Versions struct {
	versions []Object
}

// Result stores the final entity state retrieved from Store upon running
// the instructions provided by Query.
type Result struct {
	Id       string
	Kind     string
	Columns  map[string]*Versions
	Children []*Result
}

type runResult struct {
	Result *Result
	Err    error
}

func (v *Versions) add(o Object) {
	if len(v.versions) > 0 {
		i := len(v.versions) - 1
		if v.versions[i].NanoTs > o.NanoTs {
			// unsorted list. Gocrud code is doing something wrong.
			log.Fatal("Appending an object with lower ts to a sorted list")
		}
	}
	v.versions = append(v.versions, o)
}

func (v Versions) Latest() Object {
	if len(v.versions) == 0 {
		return Object{}
	}
	i := len(v.versions) - 1
	return v.versions[i]
}

func (v Versions) Oldest() Object {
	if len(v.versions) == 0 {
		return Object{}
	}
	return v.versions[0]
}

func (v Versions) Count() int {
	return len(v.versions)
}

// Retrieve the parent id for given entity id. Return ErrNoParent if parent is
// not present. Otherwise, if an error occurs during retrieval, returns that.
func Parent(id string) (parentid string, rerr error) {
	its, err := Get().GetEntity(id)
	if err != nil {
		x.LogErr(log, err).WithField("id", id).Error("While retrieving entity")
		return "", err
	}
	for _, it := range its {
		if it.Predicate == "_parent_" {
			return it.ObjectId, nil
		}
	}
	return "", ErrNoParent
}

// NewQuery is the main entrypoint to data store queries. Returns back a Query
// object pointer, to run read instructions on.
func NewQuery(id string) *Query {
	q := new(Query)
	q.id = id
	return q
}

// UptoDepth specifies the number of levels of descendants that would be
// retrieved for the entity Query points to.
//
// You can think of this in terms of a tree structure, where the Entity pointed
// to by Query points to other Entities, which in turn point to more Entities,
// and so on. For e.g. Post -> Comments -> Likes.
func (q *Query) UptoDepth(level int) *Query {
	q.maxDepth = level
	return q
}

// Collect specifies the kind of child entities to retrieve. Returns back
// a new Query pointer pointing to those children entities as a collective.
//
// Any further operations on this returned pointer would attribute to those
// children entities, and not the caller query entity.
func (q *Query) Collect(kind string) *Query {
	for _, child := range q.children {
		if child.kind == kind {
			return child
		}
	}
	child := new(Query)
	child.parent = q
	child.kind = kind
	q.children = append(q.children, child)
	return child
}

// FilterOut provides a way to well, filter out, any entities which have
// the given property.
func (q *Query) FilterOut(property string) *Query {
	if len(q.filterOut) == 0 {
		q.filterOut = make(map[string]bool)
	}
	q.filterOut[property] = true
	return q
}

func (q *Query) root() *Query {
	for q.parent != nil {
		q = q.parent
	}
	return q
}

func (q *Query) doRun(level, max int, ch chan runResult) {
	log.Debugf("Query: %+v", q)
	its, err := Get().GetEntity(q.id)
	if err != nil {
		x.LogErr(log, err).Error("While retrieving: ", q.id)
		ch <- runResult{Result: nil, Err: err}
		return
	}
	if len(its) == 0 {
		ch <- runResult{Result: new(Result), Err: nil}
		return
	}
	sort.Sort(x.Its(its))

	follow := make(map[string]*Query)
	for _, child := range q.children {
		follow[child.kind] = child
		log.WithField("kind", child.kind).WithField("child", child).Debug("Following")
	}

	result := new(Result)
	result.Columns = make(map[string]*Versions)
	it := its[0]
	result.Id = it.SubjectId
	result.Kind = it.SubjectType

	waitTimes := 0
	childChan := make(chan runResult)
	for _, it := range its {
		if it.Predicate == "_delete_" {
			// If marked as deleted, don't return this node.
			log.WithField("id", result.Id).
				WithField("kind", result.Kind).
				WithField("_delete_", true).
				Debug("Discarding due to delete bit")
			ch <- runResult{Result: new(Result), Err: nil}
			return
		}

		if it.Predicate == "_parent_" {
			log.WithField("id", result.Id).
				WithField("kind", result.Kind).
				WithField("parent", it.ObjectId).
				Debug("Not following edge back to parent")
			continue
		}

		if _, fout := q.filterOut[it.Predicate]; fout {
			log.WithField("id", result.Id).
				WithField("kind", result.Kind).
				WithField("predicate", it.Predicate).
				Debug("Discarding due to predicate filter")
			ch <- runResult{Result: new(Result), Err: nil}
			return
		}

		if len(it.ObjectId) == 0 {
			o := Object{NanoTs: it.NanoTs, Source: it.Source}
			if err := json.Unmarshal(it.Object, &o.Value); err != nil {
				x.LogErr(log, err).Error("While unmarshal")
				ch <- runResult{Result: nil, Err: err}
				return
			}

			if _, vok := result.Columns[it.Predicate]; !vok {
				result.Columns[it.Predicate] = new(Versions)
			}
			result.Columns[it.Predicate].add(o)
			continue
		}

		if childq, fw := follow[it.Predicate]; fw {
			nchildq := new(Query)
			*nchildq = *childq // This is important, otherwise id gets overwritten
			nchildq.id = it.ObjectId

			// Use child's maxDepth here, instead of parent's.
			waitTimes += 1
			log.WithField("child_id", nchildq.id).
				WithField("child_kind", nchildq.kind).Debug("Go routine for child")
			go nchildq.doRun(0, nchildq.maxDepth, childChan)
			continue
		}

		if len(it.ObjectId) > 0 && level < max {
			child := new(Query)
			child.id = it.ObjectId

			waitTimes += 1
			log.WithField("child_id", child.id).WithField("level", level+1).
				Debug("Go routine for child one level deeper")
			go child.doRun(level+1, max, childChan)
		}
	}

	// Wait for all those subroutines
	for i := 0; i < waitTimes; i++ {
		log.Debugf("Waiting for children subroutines: %v/%v", i, waitTimes-1)
		rr := <-childChan
		log.Debugf("Waiting done")
		if rr.Err != nil {
			x.LogErr(log, err).Error("While child doRun")
		} else {
			if len(rr.Result.Id) > 0 && len(rr.Result.Kind) > 0 {
				log.WithField("result", *rr.Result).Debug("Appending child")
				result.Children = append(result.Children, rr.Result)
			}
		}
	}

	ch <- runResult{Result: result, Err: nil}
	return
}

// Run finds the root from the given Query pointer, recursively executes
// the read operations, and returns back pointer to Result object.
// Any errors encountered during these stpeps is returned as well.
func (q *Query) Run() (result *Result, rerr error) {
	q = q.root()
	if len(q.id) == 0 {
		return result, errors.New("Empty entity id")
	}

	ch := make(chan runResult)
	go q.doRun(0, q.maxDepth, ch)
	rr := <-ch // Blocking wait
	return rr.Result, rr.Err
}

func (r *Result) Drop(pred string) {
	delete(r.Columns, pred)
}

func (r *Result) Debug(level int) {
	log.Debugf("Result level %v: %+v", level, r)
	for _, child := range r.Children {
		child.Debug(level + 1)
	}
}

func (r *Result) ToMap() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["id"] = r.Id
	data["kind"] = r.Kind
	var ts_latest int64
	ts_oldest := time.Now().UnixNano()
	for pred, versions := range r.Columns {
		// During conversion to JSON, to keep things simple,
		// we're dropping older versions of predicates, and
		// source and ts information across all the predicates,
		// keeping only the latest one.
		data[pred] = versions.Latest().Value
		if versions.Latest().NanoTs > ts_latest {
			ts_latest = versions.Latest().NanoTs
			data["modifier"] = versions.Latest().Source // Loss of information.
		}
		if versions.Oldest().NanoTs < ts_oldest {
			ts_oldest = versions.Oldest().NanoTs
			data["creator"] = versions.Oldest().Source
		}
	}
	data["creation_ms"] = int(ts_oldest / 1000000)
	data["modification_ms"] = int(ts_latest / 1000000) // Loss of information. Picking up latest mod time.
	kinds := make(map[string]bool)
	for _, child := range r.Children {
		kinds[child.Kind] = true
	}

	for kind := range kinds {
		var l []map[string]interface{}
		for _, child := range r.Children {
			if kind != child.Kind {
				continue
			}
			l = append(l, child.ToMap())
		}
		data[kind] = l
	}

	return
}

// ToJson creates the JSON for the data pointed by the Result pointer.
//
// Note that this doesn't find the "root" from the Result pointer, instead
// doing the processing only from the current Result pointer.
func (r *Result) ToJson() ([]byte, error) {
	data := r.ToMap()
	return json.Marshal(data)
}

// WriteJsonResponse does the same as ToJson. But also writes the JSON
// generated to http.ResponseWriter. In case of error, writes that error
// instead, in this format:
//
//  {
//   "code":    "E_ERROR",
//   "message": "error message"
//  }
func (r *Result) WriteJsonResponse(w http.ResponseWriter) {
	data, err := r.ToJson()
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	_, err = w.Write(data)
	if err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
}
