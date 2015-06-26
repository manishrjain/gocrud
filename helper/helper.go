package helper

import (
	"net/http"

	"github.com/crud/api"
	"github.com/crud/req"
	"github.com/crud/x"
)

type Entity struct {
	Id     string                 `json:"id,omitempty"`
	Kind   string                 `json:"kind,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Child  *Entity                `json:"child,omitempty"`
	Source string                 `json:"source,omitempty"`
}

type Helper struct {
	ctx *req.Context
}

func (h *Helper) SetContext(c *req.Context) {
	h.ctx = c
}

func (h *Helper) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	var e Entity
	if ok := x.ParseRequest(w, r, &e); !ok {
		return
	}
	if e.Child != nil {
		if len(e.Child.Id) > 0 {
			x.SetStatus(w, x.E_INVALID_REQUEST, "Child cannot have id specified")
			return
		}
	}
	if len(e.Id) == 0 || len(e.Kind) == 0 {
		x.SetStatus(w, x.E_INVALID_REQUEST, "No id or kind specified")
		return
	}
	if len(e.Source) == 0 {
		x.SetStatus(w, x.E_INVALID_REQUEST, "No source specified")
		return
	}

	n := api.Get(e.Kind, e.Id).SetSource(e.Source)
	for key, val := range e.Data {
		n.Set(key, val)
	}
	if e.Child != nil {
		c := n.AddChild(e.Child.Kind)
		if len(e.Child.Source) > 0 {
			c.SetSource(e.Child.Source)
		}
		for key, val := range e.Child.Data {
			c.Set(key, val)
		}
	}
	if err := n.Execute(h.ctx); err != nil {
		x.SetStatus(w, x.E_ERROR, err.Error())
		return
	}
	x.SetStatus(w, x.E_OK, "Stored")
}
