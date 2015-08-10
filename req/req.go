// This package is the initialization point for the api.
// In particular, in your init (/main) function, the flow
// is to create a req.Context and fill in required options.
// Setting table prefix, length for unique strings generated
// to assign to new entities, and setting the storage system.
package req

import (
	"sync"

	"github.com/manishrjain/gocrud/search"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("req")

type Context struct {
	NumCharsUnique int // 62^num unique strings
	Indexer        search.Indexer
	updates        chan x.Entity
	wg             *sync.WaitGroup
}

func (c *Context) processChannel() {
	defer c.wg.Done()
	for entity := range c.updates {
		doc := c.Indexer.Regenerate(entity)
		log.WithField("doc", doc).Debug("Regenerated doc")
		if search.Get() == nil {
			continue
		}
		err := search.Get().Update(doc)
		if err != nil {
			x.LogErr(log, err).WithField("doc", doc).Error("While updating doc")
		}
	}
	log.Info("Finished processing")
}

func (c *Context) RunIndexer(numRoutines int) {
	if numRoutines <= 0 {
		log.WithField("num_routines", numRoutines).
			Fatal("Invalid number of goroutines for Indexer.")
		return
	}

	// Block if we have more than 1000 pending entities for update.
	c.updates = make(chan x.Entity, 1000)

	c.wg = new(sync.WaitGroup)
	// Use 2 goroutines.
	for i := 0; i < numRoutines; i++ {
		c.wg.Add(1)
		go c.processChannel()
	}
}

func (c *Context) AddToQueue(e x.Entity) {
	c.updates <- e
}

func (c *Context) WaitForIndexer() {
	log.Debug("Waiting for indexer to finish.")
	close(c.updates)
	c.wg.Wait()
}
