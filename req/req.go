// This package is the initialization point for the api.
// In particular, in your init (/main) function, the flow
// is to create a req.Context and fill in required options.
// Setting table prefix, length for unique strings generated
// to assign to new entities, and setting the storage system.
package req

import "github.com/manishrjain/gocrud/store"

type Context struct {
	TablePrefix    string
	NumCharsUnique int // 62^num unique strings
	Store          store.Store
}
