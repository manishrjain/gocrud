# Datastore support
Gocrud supports both SQL and NoSQL databases including other datastores. This is how they can be used and initialized in your code.

##### Cassandra
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/cassandra"

func main() {
	// You can use table_cassandra.cql to generate the table.
	// Arguments: ip address, keyspace, table
	store.Get().Init("192.168.59.103", "crudtest", "instructions")
}
```

##### LevelDB
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/leveldb"

func main() {
	store.Get().Init("/tmp/ldb_filename")
}
```

##### Any SQL stores (via http://golang.org/pkg/database/sql/)
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/sqlstore"

func main() {
	// Arguments: store name, connection, table name
	store.Get().Init("mysql", "root@tcp(127.0.0.1:3306)/test", "instructions")
}
```
##### PostGreSQL (thanks philips)
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/sqlstore"

func main() {
	// Arguments: store name, connection, table name
	store.Get().Init("postgres", "postgres://localhost/test?sslmode=disable", "instructions")
}
```

##### Google Datastore
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/datastore"

func main() {
	// Arguments: table prefix, google project id.
	store.Get().Init("Test-", "project-id")
}
```

##### RethinkDB (thanks dancannon)
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/rethinkdb"

func main() {
	// Arguments: IP address with port, database, tablename
	store.Get().Init("192.168.59.103:28015", "test", "instructions")
}
```

##### MongoDB (thanks wolfeidau)
```go
import "github.com/manishrjain/gocrud/store"
import _ "github.com/manishrjain/gocrud/drivers/mongodb"

func main() {
	// Arguments: IP address with port, database, tablename
	store.Get().Init("192.168.59.103:27017", "crudtest", "instructions")
}
```

##### _Any others as requested_
Drivers for any other data stores can be easily added by implementing the `Store` interface below.

```go
type Store interface {
  Init(args ...string)
  Commit(its []*x.Instruction) error
  IsNew(subject string) bool
  GetEntity(subject string) ([]x.Instruction, error)
}
```

The data is stored in a flat “tuple” format, to allow for horizontal scaling across machines in both SQL and NoSQL databases. Again, no data is ever deleted (**Retention** principle), to allow for log tracking all the changes.


