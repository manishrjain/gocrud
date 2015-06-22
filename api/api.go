package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/crud/req"
	"github.com/crud/x"
)

var log = x.Log("api")

func Handle(w http.ResponseWriter, r *http.Request, c *req.Context) {
	var i x.Instruction
	if ok := x.ParseRequest(w, r, &i); !ok {
		return
	}
	if len(i.SubjectId) == 0 || len(i.Predicate) == 0 ||
		i.Operation == x.NOOP || len(i.Source) == 0 || len(i.SubjectType) == 0 {
		x.SetStatus(w, x.E_MISSING_REQUIRED, "Missing required fields")
		return
	}

	log.WithField("instr", i).Debug("Got instruction. Storing...")
	if ok := c.Store.Commit(c.TablePrefix, i); !ok {
		log.Error("Store failed")
		x.SetStatus(w, x.E_ERROR, "Store failed")
		return
	}
	log.Debug("Stored")
	x.SetStatus(w, x.E_OK, "Stored")
}

type Node struct {
	kind      string
	id        string
	source    string
	children  []*Node
	parent    *Node
	edges     map[string]interface{}
	Timestamp int64
}

func Get(kind, id string) *Node {
	log.WithFields(logrus.Fields{
		"func": "GetNode",
		"kind": kind,
		"id":   id,
	}).Debug("Called Get")
	n := new(Node)
	n.kind = kind
	n.id = id
	n.Timestamp = time.Now().UnixNano()
	return n
}

func (n *Node) SetSource(source string) *Node {
	n.source = source
	return n
}

func (n *Node) AddChild(kind string) *Node {
	log.WithField("childkind", kind).Debug("AddChild")
	child := new(Node)
	child.parent = n
	child.kind = kind
	child.Timestamp = n.Timestamp
	child.source = n.source
	n.children = append(n.children, child)
	return child
}

func (n *Node) Set(property string, value interface{}) *Node {
	log.WithField(property, value).Debug("SetText")
	if n.edges == nil {
		n.edges = make(map[string]interface{})
	}
	n.edges[property] = value
	return n
}

func (n *Node) recPrint(l int) {
	log.Printf("Node[%d]: %+v", l, n)
	for _, child := range n.children {
		child.recPrint(l + 1)
	}
}

func (n *Node) root() *Node {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

func (n *Node) Print() *Node {
	n = n.root()
	n.recPrint(0)
	return n
}

func (n *Node) doExecute(c *req.Context) error {
	for pred, val := range n.edges {
		if len(n.source) == 0 {
			return errors.New(fmt.Sprintf(
				"No source specified for id: %v kind: %v", n.id, n.kind))
		}

		var i x.Instruction
		i.Operation = x.ADD
		i.SubjectId = n.id
		i.SubjectType = n.kind
		i.Predicate = pred

		if b, err := json.Marshal(val); err != nil {
			return err
		} else {
			i.Object = b
		}
		i.Source = n.source
		i.NanoTs = n.Timestamp
		log.WithField("instruction", i).Debug("Committing")
		if ok := c.Store.Commit(c.TablePrefix, i); !ok {
			log.WithField("node", n).Error("Committing")
			return errors.New("While commiting node")
		} else {
			log.WithField("id", n.id).Debug("Committed")
		}
	}

	if len(n.children) == 0 {
		return nil
	}
	if len(n.source) == 0 {
		return errors.New(fmt.Sprintf(
			"No source specified for id: %v kind: %v", n.id, n.kind))
	}

	for _, child := range n.children {
		if len(child.id) == 0 {
			// Child id should be empty for all the current cases.
			for {
				child.id = x.UniqueString(5)
				log.WithField("id", child.id).Debug("Checking availability of new id")
				if isnew := c.Store.IsNew(c.TablePrefix, child.id); isnew {
					log.WithField("id", child.id).Debug("New id available")
					break
				}
			}
			// Create edge from parent to child
			var i x.Instruction
			i.Operation = x.ADD
			i.SubjectId = n.id
			i.SubjectType = n.kind
			i.Predicate = child.kind
			i.ObjectId = child.id
			i.Source = n.source
			i.NanoTs = n.Timestamp
			log.WithField("instruction", i).Debug("Committing")
			if ok := c.Store.Commit(c.TablePrefix, i); !ok {
				log.WithField("child_node", child).Error("Committing")
				return errors.New("While commiting child")
			} else {
				log.WithField("id", n.id).WithField("child_id", child.id).Debug("Committed")
			}
		}
		if err := child.doExecute(c); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Execute(c *req.Context) error {
	n = n.root()
	return n.doExecute(c)
}

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
