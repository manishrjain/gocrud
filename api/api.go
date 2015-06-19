package api

import (
	"net/http"

	"github.com/crud/req"
	"github.com/crud/x"
)

var log = x.Log("api")

func Handle(w http.ResponseWriter, r *http.Request, c *req.Context) {
	var i x.Instruction
	if ok := x.ParseRequest(w, r, &i); !ok {
		return
	}
	if len(i.Subject) == 0 || len(i.Predicate) == 0 || len(i.Object) == 0 ||
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
