// Package cassandra contains Cassandra driver for Gocrud.
// Testing of this package, works best via linux, unless you have
// cassandra tools installed on your Mac.
// To test this cassandra integration, run cassandra on docker
// $ docker pull poklet/cassandra
// $ docker run --detach --name cassone poklet/cassandra
// Now copy the contents of table_cassandra.cql to clipboard.
// $ docker run -it --rm --net container:cassone poklet/cassandra cqlsh
// Paste the cql instructions. This would generate the 'instructions'
// table in a 'crudtest' keyspace.
//
// Cassandra driver can now be imported, and initialized in social.go,
// or any other client.
// import _ "github.com/manishrjain/gocrud/drivers/cassandra"
// Initialize in main():
// store.Get().Init("cassone", "crudtest", "instructions")
package cassandra

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
)

var log = x.Log("cassandra")

type Cassandra struct {
	session *gocql.Session
}

var kIsNew, kInsert, kSelect, kScan string

func (cs *Cassandra) SetSession(session *gocql.Session) {
	cs.session = session
}

func (cs *Cassandra) Init(args ...string) {
	if len(args) != 3 && len(args) != 5 {
		log.WithField("args", args).Fatal("Invalid arguments")
		return
	}

	ipaddr := args[0]
	keyspace := args[1]
	tablename := args[2]

	cluster := gocql.NewCluster(ipaddr)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	if len(args) == 5 {
		log.WithField("username", args[3]).
			Debug("Passing username and password to Cassandra")
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: args[3],
			Password: args[4],
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		x.LogErr(log, err).Fatal("While creating session")
		return
	}
	cs.session = session

	kIsNew = fmt.Sprintf("select subject_id from %s where subject_id = ?", tablename)
	kInsert = fmt.Sprintf(`insert into %s (ts, subject_id, subject_type, predicate,
object, object_id, nano_ts, source) values (now(), ?, ?, ?, ?, ?, ?, ?)`, tablename)
	kSelect = fmt.Sprintf(`select subject_id, subject_type, predicate, object,
object_id, nano_ts, source from %s where subject_id = ?`, tablename)
	kScan = fmt.Sprintf(`select subject_type, subject_id
from %s where token(subject_id) > token(?) limit ?`, tablename)
}

func (cs *Cassandra) IsNew(subject string) bool {
	iter := cs.session.Query(kIsNew, subject).Iter()
	var sid string
	isnew := true
	for iter.Scan(&sid) {
		isnew = false
	}
	if err := iter.Close(); err != nil {
		x.LogErr(log, err).Error("While closing iterator")
		return false
	}
	return isnew
}

func (cs *Cassandra) Commit(its []*x.Instruction) error {
	b := cs.session.NewBatch(gocql.LoggedBatch)
	for _, it := range its {
		b.Query(kInsert, it.SubjectId, it.SubjectType, it.Predicate,
			it.Object, it.ObjectId, it.NanoTs, it.Source)
	}
	if err := cs.session.ExecuteBatch(b); err != nil {
		x.LogErr(log, err).Error("While executing batch")
	}
	log.WithField("len", len(its)).Debug("Stored instructions")
	return nil
}

func (cs *Cassandra) GetEntity(subject string) (
	result []x.Instruction, rerr error) {
	iter := cs.session.Query(kSelect, subject).Iter()
	var i x.Instruction
	for iter.Scan(&i.SubjectId, &i.SubjectType, &i.Predicate, &i.Object,
		&i.ObjectId, &i.NanoTs, &i.Source) {
		result = append(result, i)
	}
	if err := iter.Close(); err != nil {
		x.LogErr(log, err).Error("While iterating")
		return result, err
	}
	return result, nil
}

func (cs *Cassandra) Iterate(fromId string, num int,
	ch chan x.Entity) (rnum int, rlast x.Entity, rerr error) {

	iter := cs.session.Query(kScan, fromId, num).Iter()
	var e x.Entity
	handled := make(map[x.Entity]bool)
	rnum = 0
	for iter.Scan(&e.Kind, &e.Id) {
		rlast = e
		if _, present := handled[e]; present {
			continue
		}
		ch <- e
		rnum += 1
		handled[e] = true
		if rnum >= num {
			break
		}
	}
	if err := iter.Close(); err != nil {
		x.LogErr(log, err).Error("While closing iterator")
		return rnum, rlast, err
	}
	return rnum, rlast, nil
}

func init() {
	log.Info("Initing cassandra")
	store.Register("cassandra", new(Cassandra))
}
