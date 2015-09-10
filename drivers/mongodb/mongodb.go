package mongodb

// To test this mongodb integration, run mongodb in docker
// docker run -d --name mongo -p 27017:27017 mongo:latest
// If on Mac, find the IP address of the docker host
// $ boot2docker ip
// 192.168.59.103
// For linux it's 127.0.0.1.

import (
	"gopkg.in/manishrjain/gocrud.v1/store"
	"gopkg.in/manishrjain/gocrud.v1/x"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var log = x.Log("mongodb")

// MongoDB store backed by MongoDB
type MongoDB struct {
	session    *mgo.Session
	database   string
	collection string
}

// Init setup a new collection using the name provided
func (mdb *MongoDB) Init(args ...string) {
	if len(args) != 3 {
		log.WithField("args", args).Fatal("Invalid arguments")
		return
	}

	ipaddr := args[0]
	session, err := mgo.Dial(ipaddr)
	if err != nil {
		x.LogErr(log, err).Fatal("While dialing")
		return
	}
	session.SetMode(mgo.Monotonic, true)
	mdb.session = session
	mdb.database = args[1]
	mdb.collection = args[2]
	log.Debug("Mongodb registered")
}

// Commit inserts the instructions into the collection as documents
func (mdb *MongoDB) Commit(its []*x.Instruction) error {
	c := mdb.session.DB(mdb.database).C(mdb.collection)

	for _, i := range its {
		err := c.Insert(i)
		if err != nil {
			x.LogErr(log, err).Error("While executing batch")
			return nil
		}
	}

	log.WithField("inserted", len(its)).Debug("Stored instructions")

	return nil
}

// IsNew checks if the supplied subject identifier exists in the collection
func (mdb *MongoDB) IsNew(subject string) bool {
	c := mdb.session.DB(mdb.database).C(mdb.collection)

	i, err := c.Find(bson.M{"subjectid": subject}).Count()

	if err != nil {
		x.LogErr(log, err).Error("While running query")
		return false
	}

	if i == 0 {
		return true
	}

	return false
}

// GetEntity retrieves all documents matching the subject identifier
func (mdb *MongoDB) GetEntity(subject string) (result []x.Instruction, err error) {
	c := mdb.session.DB(mdb.database).C(mdb.collection)

	err = c.Find(bson.M{"subjectid": subject}).All(&result)
	if err != nil {
		x.LogErr(log, err).Error("While running query")
	}

	return result, err
}

func (mdb *MongoDB) Iterate(fromId string, num int, ch chan x.Entity) (found int, last x.Entity, err error) {
	log.Fatal("Not implemented")
	return
}

func init() {
	log.Info("Registering mongodb")
	store.Register("mongodb", new(MongoDB))
}
