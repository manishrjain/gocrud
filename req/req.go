package req

import "github.com/crud/store"

type Context struct {
	TablePrefix string
	Store       store.Store
}
