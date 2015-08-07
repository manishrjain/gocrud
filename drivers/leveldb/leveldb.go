package leveldb

import (
	"errors"
	"fmt"

	"github.com/manishrjain/gocrud/store"
	"github.com/manishrjain/gocrud/x"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var log = x.Log("store")

type Leveldb struct {
	db  *leveldb.DB
	opt *opt.Options
}

func (l *Leveldb) SetBloomFilter(bits int) {
	l.opt = &opt.Options{
		Filter: filter.NewBloomFilter(bits),
	}
}

func (l *Leveldb) Init(_ string, filepath string) {
	var err error
	l.db, err = leveldb.OpenFile(filepath, l.opt)
	if err != nil {
		x.LogErr(log, err).Fatal("While opening leveldb")
		return
	}
}

func (l *Leveldb) IsNew(id string) bool {
	slice := util.BytesPrefix([]byte(id))
	iter := l.db.NewIterator(slice, nil)
	isnew := true
	lg := log.WithField("id", id)
	for iter.Next() {
		if iter.Key != nil {
			isnew = false
			lg.WithField("key", string(iter.Key())).Debug("Found key")
		} else {
			lg.Debug("Found nil key")
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		x.LogErr(lg, err).Error("While iterating")
		return false
	}
	return isnew
}

func (l *Leveldb) Commit(its []*x.Instruction) error {
	var keys []string
	for _, it := range its {
		var key string
		for m := 0; m < 10; m++ {
			key = fmt.Sprintf("%s_%s", it.SubjectId, x.UniqueString(5))
			log.WithField("key", key).Debug("Checking existence of key")
			if has, err := l.db.Has([]byte(key), nil); err != nil {
				x.LogErr(log, err).WithField("key", key).Error("While check if key exists")
				continue
			} else if has {
				continue
			} else {
				break
			}
			log.Errorf("Exhausted %d tries", m)
			return errors.New("Exhausted tries")
		}
		log.WithField("key", key).Debug("Is unique")
		keys = append(keys, key)
	}

	b := new(leveldb.Batch)
	for idx, it := range its {
		key := []byte(keys[idx])
		buf, err := it.GobEncode()
		if err != nil {
			x.LogErr(log, err).Error("While encoding")
			return err
		}
		b.Put(key, buf)
	}
	if err := l.db.Write(b, nil); err != nil {
		x.LogErr(log, err).Error("While writing to db")
		return err
	}
	log.Debugf("%d instructions committed", len(its))

	return nil
}

func (l *Leveldb) GetEntity(id string) (result []x.Instruction, rerr error) {
	slice := util.BytesPrefix([]byte(id))
	iter := l.db.NewIterator(slice, nil)
	for iter.Next() {
		buf := iter.Value()
		if buf == nil {
			break
		}
		var i x.Instruction
		if err := i.GobDecode(buf); err != nil {
			x.LogErr(log, err).Error("While decoding")
			return result, err
		}
		result = append(result, i)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		x.LogErr(log, err).Error("While iterating")
	}
	return result, err
}

func init() {
	log.Info("Initing leveldb")
	l := new(Leveldb)
	l.SetBloomFilter(13)
	store.Register("leveldb", l)
}
