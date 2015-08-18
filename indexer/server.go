package indexer

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/manishrjain/gocrud/search"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

// Incremental indexing server to continously regenerate
// and index entities to keep store and search in-sync.
type Server struct {
	ch chan x.Entity
}

// NewServer returns back a server which runs continously in
// a loop to find and re-index entities stored.
// You can control the amount of memory consumed by the server
// via buffer of pending entities in the channel, and the
// rate of processing of these entities via numRoutines.
func NewServer(buffer int, numRoutines int) *Server {
	if search.Get() == nil {
		log.Fatal("No search engine found")
	}
	s := new(Server)
	s.ch = make(chan x.Entity, buffer)
	for i := 0; i < numRoutines; i++ {
		go s.regenerateAndIndex()
	}
	return s
}

func (s *Server) regenerateAndIndex() {
	for entity := range s.ch {
		idxr, ok := Get(entity.Kind)
		if !ok {
			continue
		}

		doc := idxr.Regenerate(entity)
		log.WithField("doc", doc).Debug("Regenerated doc")
		if err := search.Get().Update(doc); err != nil {
			x.LogErr(log, err).WithField("doc", doc).
				Error("While updating in search engine")
		}
	}
}

func (s *Server) cycleOnce() {
	var total uint64
	from := ""
	for {
		found, last, err := store.Get().Iterate(from, 1000, s.ch)
		if err != nil {
			x.LogErr(log, err).Error("While iterating")
			return
		}
		if found == 0 {
			log.WithField("total", total).Info("Reached end of cycle")
			return
		}
		log.WithFields(logrus.Fields{
			"num_processed": found,
			"last":          last,
		}).Debug("Iteration chunk done")
		total += uint64(found)
		from = last.Id
	}
	log.Fatal("This should never be reached.")
	return
}

// InfiniteLoop would infinitely cycle over all entities in the
// store, waiting for wait duration after each cycle.
func (s *Server) InfiniteLoop(wait time.Duration) {
	for {
		s.cycleOnce()
		log.Debug("Sleeping...")
		time.Sleep(wait)
	}
}
