// api package provides the CRUD apis for data manipulation.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("api")

// Node stores the create and update instructions, acting as the modifier
// to the entity Node relates to.
type Node struct {
	kind      string
	id        string
	source    string
	children  []*Node
	parent    *Node
	edges     map[string]interface{}
	Timestamp int64
}

// Get is the main entrypoint to updates. Returns back a Node
// object pointer, to run create and update operations on.
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

// SetSource sets the author of the update. Generally, the userid of the
// modifier.
func (n *Node) SetSource(source string) *Node {
	n.source = source
	return n
}

// AddChild creates a new entity with the given kind, and creates a
// directed relationship from current entity to this new child entity.
// This is useful to generate arbitrarily deep and complex data structures,
// for e.g. Posts by Users, Comments on Posts, Likes on Posts etc.
//
// Retuns the Node pointer for the child entity, so any update operations
// done on this pointer would be reflected in the child entity.
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

// Set allows you to set the property and value on the current entity.
// This would effectively replace any other value this property had,
// on this entity node pointer represents.
func (n *Node) Set(property string, value interface{}) *Node {
	log.WithField(property, value).Debug("Set")
	if n.edges == nil {
		n.edges = make(map[string]interface{})
	}
	n.edges[property] = value
	return n
}

// Marks the current entity for deletion. This is equivalent to doing a
// Set("delete", true), and then running q.FilterOut("delete") during
// query phase.
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

// Print finds the root from the given Node pointer, and does a recursive
// print on the tree for debugging purposes.
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

// Execute finds the root from the given Node pointer, recursively generates
// the set of instructions to store, and commits them. Returns any errors
// encountered during these steps.
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
