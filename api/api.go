package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

func (n *Node) toJson(data *map[string]interface{}) error {
	return nil
}

func (n *Node) JsonGraph(c *req.Context) (rjson []byte, rerr error) {
	n = n.root()

	its, err := c.Store.GetEntity(c.TablePrefix, n.id)
	if err != nil {
		x.LogErr(log, err).Error("While retrieving ", n.id)
		return
	}
	log.Infof("Got data: %+v", its)
	sort.Sort(x.Its(its))

	// TODO: Put the flat data back into a graph.
	var data map[string]interface{}
	for _, i := range its {
		data["source"] = i.Source
	}
	return rjson, nil
}
