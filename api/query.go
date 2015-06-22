package api

import (
	"encoding/json"
	"net/http"

	"github.com/crud/req"
	"github.com/crud/x"
)

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
type Result struct {
	Id       string
	Kind     string
	Children []*Result
	Columns  map[string]Object
}

func NewQuery(kind, id string) *Query {
	q := new(Query)
	q.kind = kind
	q.id = id
	return q
}

func (q *Query) UptoDepth(level int) *Query {
	q.maxDepth = level
	return q
}

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

func (q *Query) doRun(c *req.Context, level, max int) (result *Result, rerr error) {
	log.Debugf("Query: %+v", q)
	its, err := c.Store.GetEntity(c.TablePrefix, q.id)
	if err != nil {
		x.LogErr(log, err).Error("While retrieving: ", q.id)
		return nil, err
	}
	if len(its) == 0 {
		return new(Result), nil
	}

	follow := make(map[string]*Query)
	for _, child := range q.children {
		follow[child.kind] = child
	}

	result = new(Result)
	result.Columns = make(map[string]Object)
	it := its[0]
	result.Id = it.SubjectId
	result.Kind = it.SubjectType

	for _, it := range its {
		if _, fout := q.filterOut[it.Predicate]; fout {
			log.WithField("id", result.Id).
				WithField("predicate", it.Predicate).
				Debug("Discarding due to predicate filter")
			return new(Result), nil
		}

		if len(it.ObjectId) == 0 {
			o := Object{NanoTs: it.NanoTs, Source: it.Source}
			if err := json.Unmarshal(it.Object, &o.Value); err != nil {
				x.LogErr(log, err).Error("While unmarshal")
				return nil, err
			}

			result.Columns[it.Predicate] = o
			continue
		}

		if child, fw := follow[it.Predicate]; fw {
			child.id = it.ObjectId
			if cr, err := child.doRun(c, level+1, max); err == nil {
				if len(cr.Id) > 0 && len(cr.Kind) > 0 {
					result.Children = append(result.Children, cr)
				}
			} else {
				x.LogErr(log, err).Error("While doRun")
			}
			continue
		}

		if len(it.ObjectId) > 0 && level < max {
			child := new(Query)
			child.id = it.ObjectId

			if cr, err := child.doRun(c, level+1, max); err == nil {
				result.Children = append(result.Children, cr)
			} else {
				x.LogErr(log, err).Error("While doRun")
			}
		}
	}

	return result, nil
}

func (q *Query) Run(c *req.Context) (result *Result, rerr error) {
	q = q.root()

	return q.doRun(c, 0, q.maxDepth)
}

func (r *Result) Debug(level int) {
	log.Debugf("Result level %v: %+v", level, r)
	for _, child := range r.Children {
		child.Debug(level + 1)
	}
}

func (r *Result) toJson() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["id"] = r.Id
	data["kind"] = r.Kind
	var ts int64
	for k, o := range r.Columns {
		data[k] = o.Value
		data["source"] = o.Source // Loss of information.
		if o.NanoTs > ts {
			ts = o.NanoTs
		}
	}
	data["ts_millis"] = int(ts / 1000000) // Loss of information. Picking up latest mod time.
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
			l = append(l, child.toJson())
		}
		data[kind] = l
	}

	return
}

func (r *Result) ToJson() ([]byte, error) {
	data := r.toJson()
	return json.Marshal(data)
}

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
