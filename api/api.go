package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gocrud/req"
	"github.com/gocrud/x"
)

var log = x.Log("api")

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
	log.WithField(property, value).Debug("Set")
	if n.edges == nil {
		n.edges = make(map[string]interface{})
	}
	n.edges[property] = value
	return n
}

func (n *Node) MarkDeleted() *Node {
	return n.Set("_delete_", true)
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

func (n *Node) doExecute(c *req.Context, its *[]*x.Instruction) error {
	for pred, val := range n.edges {
		if len(n.source) == 0 {
			return errors.New(fmt.Sprintf(
				"No source specified for id: %v kind: %v", n.id, n.kind))
		}

		i := new(x.Instruction)
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
		log.WithField("instruction", i).Debug("Pushing to list")
		*its = append(*its, i)
	}

	if len(n.children) == 0 {
		return nil
	}
	if len(n.source) == 0 {
		return errors.New(fmt.Sprintf(
			"No source specified for id: %v kind: %v", n.id, n.kind))
	}

	// Children can only be added, not deleted via API. But they can be stopped
	// from being retrieved.
	// Scenario: How do I stop childA from being retrieved?
	// Answer:
	// Modify child by adding a 'deleted' edge
	// Get(ChildKind, ChildId).Set("deleted", true).Execute(c)
	//
	// Then for retrieval from parent:
	// NewQuery(ParentKind, ParentId).Collect(ChildKind).FilterOut("deleted")
	// This would remove all children with a 'deleted' edge.

	for _, child := range n.children {
		if len(child.id) > 0 {
			log.WithField("child_id", child.id).Fatal(
				"Child id should be empty for all current use cases")
			return errors.New("Non empty child id")
		}

		for idx := 0; ; idx++ {
			child.id = x.UniqueString(5)
			log.WithField("id", child.id).Debug("Checking availability of new id")
			if isnew := c.Store.IsNew(c.TablePrefix, child.id); isnew {
				log.WithField("id", child.id).Debug("New id available")
				break
			}
			if idx >= 30 {
				return errors.New("Unable to find new id")
			}
		}
		// Create edge from parent to child
		i := new(x.Instruction)
		i.SubjectId = n.id
		i.SubjectType = n.kind
		i.Predicate = child.kind
		i.ObjectId = child.id
		i.Source = n.source
		i.NanoTs = n.Timestamp
		log.WithField("instruction", i).Debug("Pushing to list")
		*its = append(*its, i)
		if err := child.doExecute(c, its); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Execute(c *req.Context) error {
	n = n.root()

	var its []*x.Instruction
	err := n.doExecute(c, &its)
	if err != nil {
		return err
	}
	if len(its) == 0 {
		return errors.New("No instructions generated")
	}

	return c.Store.Commit(c.TablePrefix, its)
}
