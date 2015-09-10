package rethinkdb

// To test this rethinkdb integration, run rethinkdb on docker
// docker run -d --name rethinkdb -p 28015:28015 -p 8080:8080 rethinkdb:latest
// If on Mac, find the IP address of the docker host
// $ boot2docker ip
// 192.168.59.103
// For linux it's 127.0.0.1.
// Now you can go to 192.168.59.103:8080 (or 127.0.0.1:8080) and create
// table 'instructions'. Once created, create an index by going to
// 'Data Explorer', and running this:
// r.db('test').table('instructions').indexCreate('SubjectId')

import (
	r "github.com/dancannon/gorethink"
	"gopkg.in/manishrjain/gocrud.v1/store"
	"gopkg.in/manishrjain/gocrud.v1/x"
)

var log = x.Log("rethinkdb")

type RethinkDB struct {
	session *r.Session
	table   string
}

func (rdb *RethinkDB) SetSession(session *r.Session) {
	rdb.session = session
}

func (rdb *RethinkDB) Init(args ...string) {
	if len(args) != 3 {
		log.WithField("args", args).Fatal("Invalid arguments")
		return
	}

	ipaddr := args[0]
	dbname := args[1]
	tablename := args[2]
	session, err := r.Connect(r.ConnectOpts{
		// Address:  "192.168.59.103:28015",
		Address:  ipaddr,
		Database: dbname,
	})
	if err != nil {
		x.LogErr(log, err).Fatal("While connecting")
		return
	}
	rdb.session = session
	rdb.table = tablename
}

func (rdb *RethinkDB) IsNew(subject string) bool {
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

func (rdb *RethinkDB) Commit(its []*x.Instruction) error {
	res, err := r.Table(rdb.table).Insert(its).RunWrite(rdb.session)
	if err != nil {
		x.LogErr(log, err).Error("While executing batch")
		return nil
	}

	log.WithField("inserted", res.Inserted+res.Replaced).Debug("Stored instructions")
	return nil
}

func (rdb *RethinkDB) GetEntity(subject string) (
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

	return result, nil
}

func (rdb *RethinkDB) Iterate(fromId string, num int, ch chan x.Entity) (found int, last x.Entity, err error) {
	log.Fatal("Not implemented")
	return
}

func init() {
	log.Info("Registering rethinkdb")
	store.Register("rethinkdb", new(RethinkDB))
}
