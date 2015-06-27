package req

import "github.com/gocrud/store"

type Context struct {
	TablePrefix string
	Store       store.Store
}
