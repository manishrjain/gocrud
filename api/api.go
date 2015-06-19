package api

import (
	"errors"
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

type op struct {
	id   string
	text string
}

type Node struct {
	kind      string
	id        string
	source    string
	child     *Node
	parent    *Node
	edges     map[string]op
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
	n.child = new(Node)
	n.child.parent = n
	n.child.kind = kind
	n.child.Timestamp = n.Timestamp
	return n.child
}

func (n *Node) SetText(property, value string) *Node {
	log.WithField(property, value).Debug("SetText")
	if n.edges == nil {
		n.edges = make(map[string]op)
	}
	o, present := n.edges[property]
	if !present {
		o = op{text: value}
	} else {
		o.text = value
	}
	n.edges[property] = o
	return n
}

func (n *Node) SetId(property, id string) *Node {
	if n.edges == nil {
		n.edges = make(map[string]op)
	}
	o, present := n.edges[property]
	if !present {
		o = op{id: id}
	} else {
		o.id = id
	}
	n.edges[property] = o
	return n
}

func (n *Node) recPrint(l int) {
	log.Printf("Node[%d]: %+v", l, n)
	if n.child != nil {
		n.child.recPrint(l + 1)
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
	for pred, op := range n.edges {
		var i x.Instruction
		i.Operation = x.ADD
		i.SubjectId = n.id
		i.SubjectType = n.kind
		i.Predicate = pred
		i.ObjectText = op.text
		i.ObjectId = op.id
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

	if n.child == nil {
		return nil
	}

	if len(n.child.id) == 0 {
		// Child id should be empty for all the current cases.
		for {
			n.child.id = x.UniqueString(5)
			log.WithField("id", n.child.id).Debug("Checking availability of new id")
			if isnew := c.Store.IsNew(c.TablePrefix, n.child.kind, n.child.id); isnew {
				log.WithField("id", n.child.id).Debug("New id available")
				break
			}
		}
		// Create edge from parent to child
		var i x.Instruction
		i.Operation = x.ADD
		i.SubjectId = n.id
		i.SubjectType = n.kind
		i.Predicate = n.child.kind
		i.ObjectId = n.child.id
		i.Source = n.child.source
		i.NanoTs = n.Timestamp
		log.WithField("instruction", i).Debug("Committing")
		if ok := c.Store.Commit(c.TablePrefix, i); !ok {
			log.WithField("child_node", n.child).Error("Committing")
			return errors.New("While commiting child")
		} else {
			log.WithField("id", n.id).WithField("child_id", n.child.id).Debug("Committed")
		}
	}
	return n.child.doExecute(c)
}

func (n *Node) Execute(c *req.Context) error {
	n = n.root()
	return n.doExecute(c)
}
