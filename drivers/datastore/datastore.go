package datastore

import (
	"gopkg.in/manishrjain/gocrud.v1/store"
	"gopkg.in/manishrjain/gocrud.v1/x"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

var log = x.Log("store")

type Datastore struct {
	tablePrefix string
	ctx         context.Context
	projectId   string
}

// Init takes 2 arguments: tablePrefix and projectId.
func (ds *Datastore) Init(args ...string) {
	if len(args) != 2 {
		log.WithField("args", args).Fatal("Invalid arguments")
		return
	}

	ds.tablePrefix = args[0]
	project := args[1]

	client, err := google.DefaultClient(oauth2.NoContext,
		"https://www.googleapis.com/auth/devstorage.full_control")
	if err != nil {
		x.LogErr(log, err).Fatal("Unable to get client")
	}
	ds.ctx = cloud.NewContext(project, client)
	if ds.ctx == nil {
		log.Fatal("Failed to get context. context is nil")
	}
	ds.projectId = project
	log.Info("Connection to Google datastore established")
}

func (ds *Datastore) getIKey(i x.Instruction) *datastore.Key {
	skey := datastore.NewKey(ds.ctx, ds.tablePrefix+"Entity", i.SubjectId, 0, nil)
	return datastore.NewIncompleteKey(ds.ctx, ds.tablePrefix+"Instruction", skey)
}

func (ds *Datastore) Commit(its []*x.Instruction) error {
	var keys []*datastore.Key
	for _, i := range its {
		dkey := ds.getIKey(*i)
		keys = append(keys, dkey)
	}
	client, err := datastore.NewClient(ds.ctx, ds.projectId)
	if err != nil {
		x.LogErr(log, err).Error("While creating new client")
		return err
	}
	if _, err := client.PutMulti(ds.ctx, keys, its); err != nil {
		x.LogErr(log, err).Error("While committing instructions")
		return err
	}
	log.Debugf("%d Instructions committed", len(its))
	return nil
}

func (ds *Datastore) IsNew(id string) bool {
	dkey := datastore.NewKey(ds.ctx, ds.tablePrefix+"Entity", id, 0, nil)
	client, err := datastore.NewClient(ds.ctx, ds.projectId)
	if err != nil {
		x.LogErr(log, err).Error("While creating client")
		return false
	}
	q := datastore.NewQuery(ds.tablePrefix + "Instruction").Ancestor(dkey).
		Limit(1).KeysOnly()
	keys, err := client.GetAll(ds.ctx, q, nil)
	if err != nil {
		x.LogErr(log, err).Error("While GetAll")
		return false
	}
	if len(keys) > 0 {
		return false
	}
	return true
}

func (ds *Datastore) GetEntity(subject string) (reply []x.Instruction, rerr error) {
	skey := datastore.NewKey(ds.ctx, ds.tablePrefix+"Entity", subject, 0, nil)
	log.Infof("skey: %+v", skey)

	client, err := datastore.NewClient(ds.ctx, ds.projectId)
	if err != nil {
		x.LogErr(log, err).Error("While creating client")
		return reply, err
	}

	var dkeys []*datastore.Key
	q := datastore.NewQuery(ds.tablePrefix + "Instruction").Ancestor(skey)
	dkeys, rerr = client.GetAll(ds.ctx, q, &reply)
	log.Debugf("Got num keys: %+v", len(dkeys))
	return
}

func (ds *Datastore) Iterate(fromId string, num int, ch chan x.Entity) (found int, last x.Entity, err error) {
	log.Fatal("Not implemented")
	return
}

func init() {
	log.Info("Initing datastore")
	store.Register("datastore", new(Datastore))
}
