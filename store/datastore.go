package store

import (
	"github.com/crud/x"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

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

func (ds *Datastore) getIKey(i x.Instruction, tablePrefix string) *datastore.Key {
	skey := datastore.NewKey(ds.ctx, tablePrefix+"Entity", i.SubjectId, 0, nil)
	return datastore.NewIncompleteKey(ds.ctx, tablePrefix+"Instruction", skey)
}

func (ds *Datastore) Commit(t string, its []*x.Instruction) error {
	var keys []*datastore.Key
	for _, i := range its {
		dkey := ds.getIKey(*i, t)
		keys = append(keys, dkey)
	}
	if _, err := datastore.PutMulti(ds.ctx, keys, its); err != nil {
		x.LogErr(log, err).Error("While committing instructions")
		return err
	}
	log.Debugf("%d Instructions committed", len(its))
	return nil
}

func (ds *Datastore) IsNew(t, id string) bool {
	dkey := datastore.NewKey(ds.ctx, t+"Entity", id, 0, nil)
	keys, err := datastore.NewQuery(t+"Instruction").Ancestor(dkey).
		Limit(1).KeysOnly().GetAll(ds.ctx, nil)
	if err != nil {
		return false
	}
	if len(keys) > 0 {
		return false
	}
	return true
}

func (ds *Datastore) GetEntity(t, subject string) (reply []x.Instruction, rerr error) {
	skey := datastore.NewKey(ds.ctx, t+"Entity", subject, 0, nil)
	log.Infof("skey: %+v", skey)
	dkeys, rerr := datastore.NewQuery(t+"Instruction").Ancestor(skey).GetAll(ds.ctx, &reply)
	log.Debugf("Got num keys: %+v", len(dkeys))
	return
}
