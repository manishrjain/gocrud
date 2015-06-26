package store

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

	_ "github.com/cloudflare/cfssl/log" // My goimports is going nuts
	"github.com/crud/x"
	"github.com/gocql/gocql"
)

type Cassandra struct {
	session *gocql.Session
	// table   string
}

var kIsNew, kInsert, kSelect string

func (cs *Cassandra) SetSession(session *gocql.Session) {
	cs.session = session
}

func (cs *Cassandra) Init(tablename string) {
	// cs.table = tablename
	kIsNew = fmt.Sprintf("select subject_id from %s where subject_id = ?", tablename)
	kInsert = fmt.Sprintf(`insert into %s (ts, subject_id, subject_type, predicate,
object, object_id, nano_ts, source) values (now(), ?, ?, ?, ?, ?, ?, ?)`, tablename)
	kSelect = fmt.Sprintf(`select subject_id, subject_type, predicate, object,
object_id, nano_ts, source from %s where subject_id = ?`, tablename)
}

func (cs *Cassandra) IsNew(_ string, subject string) bool {
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

func (cs *Cassandra) Commit(_ string, its []*x.Instruction) error {
	b := cs.session.NewBatch(gocql.LoggedBatch)
	for _, it := range its {
		b.Query(kInsert, it.SubjectId, it.SubjectType, it.Predicate,
			it.Object, it.ObjectId, it.NanoTs, it.Source)
	}
	if err := cs.session.ExecuteBatch(b); err != nil {
		x.LogErr(log, err).Error("While executing batch")
	}
	log.Debugf("Stored %d instructions", len(its))
	return nil
}

func (cs *Cassandra) GetEntity(_ string, subject string) (
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
