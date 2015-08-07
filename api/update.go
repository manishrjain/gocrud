// api package provides the CRUD apis for data manipulation.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/manishrjain/gocrud/req"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("api")

// Update stores the create and update instructions, acting as the modifier
// to the entity Update relates to.
type Update struct {
	kind      string
	id        string
	source    string
	children  []*Update
	parent    *Update
	edges     map[string]interface{}
	Timestamp int64
}

// NewUpdate is the main entrypoint to updates. Returns back a Update
// object pointer, to run create and update operations on.
func NewUpdate(kind, id string) *Update {
	log.WithFields(logrus.Fields{
		"func": "NewUpdate",
		"kind": kind,
		"id":   id,
	}).Debug("Called")
	n := new(Update)
	n.kind = kind
	n.id = id
	n.Timestamp = time.Now().UnixNano()
	return n
}

// SetSource sets the author of the update. Generally, the userid of the
// modifier.
func (n *Update) SetSource(source string) *Update {
	n.source = source
	return n
}

// AddChild creates a new entity with the given kind, and creates a
// directed relationship from current entity to this new child entity.
// This is useful to generate arbitrarily deep and complex data structures,
// for e.g. Posts by Users, Comments on Posts, Likes on Posts etc.
//
// Retuns the Update pointer for the child entity, so any update operations
// done on this pointer would be reflected in the child entity.
func (n *Update) AddChild(kind string) *Update {
	log.WithField("childkind", kind).Debug("AddChild")
	child := new(Update)
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
func (n *Update) Set(property string, value interface{}) *Update {
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
func (n *Update) MarkDeleted() *Update {
	return n.Set("_delete_", true)
}

func (n *Update) recPrint(l int) {
	log.Printf("Update[%d]: %+v", l, n)
	for _, child := range n.children {
		child.recPrint(l + 1)
	}
}

func (n *Update) root() *Update {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

// Print finds the root from the given Update pointer, and does a recursive
// print on the tree for debugging purposes.
func (n *Update) Print() *Update {
	n = n.root()
	n.recPrint(0)
	return n
}

func (n *Update) doExecute(c *req.Context, its *[]*x.Instruction) error {
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
			child.id = x.UniqueString(c.NumCharsUnique)
			log.WithField("id", child.id).Debug("Checking availability of new id")
			if isnew := store.Get().IsNew(child.id); isnew {
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

// Execute finds the root from the given Update pointer, recursively generates
// the set of instructions to store, and commits them. Returns any errors
// encountered during these steps.
func (n *Update) Execute(c *req.Context) error {
	if c.NumCharsUnique <= 0 {
		log.Fatal("Invalid number of chars for generating unique ids. Set req.Context.NumCharsUnique")
		return errors.New("Invalid req.Context.NumCharsUnique")
	}

	n = n.root()

	var its []*x.Instruction
	err := n.doExecute(c, &its)
	if err != nil {
		return err
	}
	if len(its) == 0 {
		return errors.New("No instructions generated")
	}

	if rerr := store.Get().Commit(its); rerr != nil {
		return rerr
	}

	{
		// This block of code figures out which entities have been modified, runs
		// OnUpdate calls on them, to then compile a list of unique entities which
		// need to be regenerated, and sends them off to the c.Updates channel.
		updates := make(map[x.Entity]bool)
		regens := make(map[x.Entity]bool)
		for _, it := range its {
			e := x.Entity{Kind: it.SubjectType, Id: it.SubjectId}
			updates[e] = true
		}
		for entity := range updates {
			dirty := c.Indexer.OnUpdate(entity)
			for _, de := range dirty {
				regens[de] = true
			}
		}

		log.WithField("num_updates", len(updates)).
			WithField("num_regenerate", len(regens)).Debug("Sending for doc regeneration")
		for entity := range regens {
			log.WithField("kind", entity.Kind).WithField("id", entity.Id).
				Debug("Send to updates channel")
			c.AddToQueue(entity)
		}
	}
	return nil
}
