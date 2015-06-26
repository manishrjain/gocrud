# crud
Go library to simplify creating, updating and deleting arbitrary depth structured data — to make building REST services fast and easy.

This library is built to allow these properties for the CRUD server:
1. **Versioning**: Keep track of all edits to the data, including deletion operations.
1. **Authorship**: Be able to track who edited (/deleted) what.
1. **Persistence**: On deletion, only mark it as deleted. Never actually delete any data.

The library makes it easy to have *Parent-Child* relationships, quite common in today’s CRUD operations. For e.g.
- Posts created by User (User -> Post)
- Comments on Posts (Post -> Comment)
- Likes on Posts (Post -> Like)
- Likes on Comments (Comment -> Like)
And be able to traverse these relationships and retrieve all of the children, grandchildren etc. For e.g. (User -> Post -> [(Comment -> Like), Like])

The library does this by utilizing Graph operations, but without using a full fledged Graph database. This means the library can be used to quickly build a Go backend to serve arbitrarily complex data, while still using your database of choice.

This library supports both SQL and NoSQL databases, namely
1. Cassandra
1. LevelDB
1. Any SQL stores (via http://golang.org/pkg/database/sql/)
1. Google Datastore
1. _Any others as requested_

In fact, it exposes a simple interface for operations requiring databases, so you can easily add your favorite database (or request for addition).

The data is stored in a flat “tuple” format, to allow for horizontal scaling across machines in both SQL and NoSQL databases. Again, no data is ever deleted, to allow for log tracking all the changes.

# Usage (with example)
### Let’s use a social backend as an example.