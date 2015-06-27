package store

// To test this rethinkdb integration, run rethinkdb on docker
// docker run -d --name rethinkdb -p 28105:28105 rethinkdb:latest
// If on Mac, find the IP address of the docker host
// $ boot2docker ip
// 192.168.59.103
// For linux it's 127.0.0.1.

import (
	r "github.com/dancannon/gorethink"
	"github.com/davecgh/go-spew/spew"
	"github.com/manishrjain/gocrud/x"
)

type RethinkDB struct {
	session *r.Session
	table   string
}

func (rdb *RethinkDB) SetSession(session *r.Session) {
	rdb.session = session
}

func (rdb *RethinkDB) Init(_ string, tablename string) {
	rdb.table = tablename
}

func (rdb *RethinkDB) IsNew(_ string, subject string) bool {
	iter, err := r.Table(rdb.table).Get(subject).Run(rdb.session)
	if err != nil {
		x.LogErr(log, err).Error("While running query")
		return false
	}

	isnew := true
	if !iter.IsNil() {
		isnew = true
	}

	if err := iter.Close(); err != nil {
		x.LogErr(log, err).Error("While closing iterator")
		return false
	}
	return isnew
}

func (rdb *RethinkDB) Commit(_ string, its []*x.Instruction) error {
	res, err := r.Table(rdb.table).Insert(its).RunWrite(rdb.session)
	if err != nil {
		x.LogErr(log, err).Error("While executing batch")
		return nil
	}

	log.WithField("inserted", res.Inserted+res.Replaced).Debug("Stored instructions")
	return nil
}

func (rdb *RethinkDB) GetEntity(_ string, subject string) (
	result []x.Instruction, rerr error,
) {
	iter, err := r.Table(rdb.table).GetAllByIndex("SubjectId", subject).Run(rdb.session)
	if err != nil {
		x.LogErr(log, err).Error("While running query")
		return result, err
	}

	err = iter.All(&result)
	if err != nil {
		x.LogErr(log, err).Error("While iterating")
		return result, err
	}

	if err := iter.Close(); err != nil {
		x.LogErr(log, err).Error("While closing iterator")
		return result, err
	}

	spew.Dump(result)

	return result, nil
}
