package store

import (
	"github.com/crud/x"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

var log = x.Log("store")

type Datastore struct {
	ctx context.Context
}

func (ds *Datastore) Init(project string) {
	client, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/devstorage.full_control")
	if err != nil {
		x.LogErr(log, err).Fatal("Unable to get client")
	}
	ds.ctx = cloud.NewContext(project, client)
	if ds.ctx == nil {
		log.Fatal("Failed to get context. context is nil")
	}
	log.Info("Connection to Google datastore established")
}

func (ds *Datastore) getObjectKey(i x.Instruction, tablePrefix string) *datastore.Key {
	skey := datastore.NewKey(ds.ctx, tablePrefix+"Subject", i.SubjectId, 0, nil)
	ekey := datastore.NewKey(ds.ctx, tablePrefix+"Predicate", i.Predicate, 0, skey)
	return datastore.NewIncompleteKey(ds.ctx, tablePrefix+"Instruction", ekey)
}

func (ds *Datastore) Commit(t string, i x.Instruction) bool {
	dkey := ds.getObjectKey(i, t)
	if i.Operation == x.NOOP {
		log.WithField("instr", i).Error("Found NOOP instruction")
		return false
	}
	if _, err := datastore.Put(ds.ctx, dkey, &i); err != nil {
		x.LogErr(log, err).WithField("instr", i).Error("While adding instruction")
		return false
	}
	// Mark Subject as dirty.
	return true
}

func (ds *Datastore) IsNew(t, kind, id string) bool {
	dkey := datastore.NewKey(ds.ctx, t+kind, id, 0, nil)
	var i x.Instruction
	if err := datastore.Get(ds.ctx, dkey, &i); err == datastore.ErrNoSuchEntity {
		return true
	}
	return false
}

/*
func (ds *Datastore) ReadEntity(t, subject string) (n x.Node, rerr error) {
	skey := datastore.NewKey(ds.ctx, t+"Subject", subject, 0, nil)
	var its []Instruction
	_, err := datastore.NewQuery(t+"Predicate").Ancestor(skey).GetAll(ds.ctx, &its)
	if err != nil {
		x.LogErr(log, err).Error("While retrieving instructions")
		return n, err
	}
	log.Info("Got data: %+v", its)
	return
}
*/
