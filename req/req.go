package req

import "github.com/manishrjain/gocrud/store"

type Context struct {
	TablePrefix string
	Store       store.Store
}
