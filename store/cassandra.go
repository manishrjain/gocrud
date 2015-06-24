package store

import (
	"github.com/crud/x"
	"github.com/gocql/gocql"
)

type Cassandra struct {
	session *gocql.Session
	table   string
}

func (cs *Cassandra) Init(tablename string) {
	cs.table = tablename
}

const kIsNew = `select subject_id from ? where subject_id = ?`

func (cs *Cassandra) IsNew(_ string, subject string) bool {
	iter := cs.session.Query(kIsNew, cs.table, subject).Iter()
	var sid
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
}

func (cs *Cassandra) GetEntity(_ string, subject string) (result []x.Instruction, rerr error) {
}
