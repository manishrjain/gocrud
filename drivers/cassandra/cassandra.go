package cassandra

// To test this cassandra integration, run cassandra on docker
// docker run -d --name cass -p 9042:9042 abh1nav/cassandra:latest
// If on Mac, find the IP address of the docker host
// $ boot2docker ip
// 192.168.59.103
// For linux it's 127.0.0.1.
// Then, on Mac connect to it with
// $ cqlsh 192.168.59.103 9042

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

var kIsNew, kInsert, kSelect string

func (cs *Cassandra) SetSession(session *gocql.Session) {
	cs.session = session
}

func (cs *Cassandra) Init(args ...string) {
	if len(args) != 3 || len(args) != 5 {
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

func init() {
	log.Info("Initing cassandra")
	store.Register("cassandra", new(Cassandra))
}
