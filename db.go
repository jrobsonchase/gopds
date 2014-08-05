package gopds

import (
	"encoding/json"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"os"
	"time"
)

type OpdsDB struct {
	path string
	dbs  map[string]*leveldb.DB
}

func OpenDB(path string) (*OpdsDB, error) {
	database := &OpdsDB{path: path}
	database.dbs = make(map[string]*leveldb.DB)
	pathInfo, err := os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, os.ModeDir|0777)
		if err != nil {
			return nil, err
		}
	} else {
		if !pathInfo.IsDir() {
			return nil, errors.New("Not a directory")
		}
	}
	return database, nil
}

func (db *OpdsDB) GetDB(database string) (*leveldb.DB, error) {
	_, exists := db.dbs[database]
	if !exists {
		newdb, err := leveldb.OpenFile(db.path+"/"+database, nil)
		if err != nil {
			return nil, err
		}
		db.dbs[database] = newdb
	}
	return db.dbs[database], nil
}

func (db *OpdsDB) Set(database, key string, value interface{}) error {
	d, err := db.GetDB(database)
	if err != nil {
		return err
	}
	jval, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = d.Put([]byte(key), jval, nil)
	if err != nil {
		return err
	}
	return nil
}

func (db *OpdsDB) Get(database, key string, dest interface{}) error {
	d, err := db.GetDB(database)
	if err != nil {
		return err
	}
	jval, err := d.Get([]byte(key), nil)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jval, dest)
	if err != nil {
		return err
	}
	return nil
}

func (db *OpdsDB) Exists(database, key string) (bool, error) {
	d, err := db.GetDB(database)
	if err != nil {
		return false, err
	}
	_, err = d.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return false, nil
	}
	if err == nil {
		return true, nil
	}
	return false, err
}

func (db *OpdsDB) NewIterator(database string) (iterator.Iterator, error) {
	d, err := db.GetDB(database)
	if err != nil {
		return nil, err
	}
	return d.NewIterator(nil, nil), nil
}

func (db *OpdsDB) Count(database string) (int, error) {
	iter, err := db.NewIterator(database)
	if err != nil {
		return 0, err
	}
	count := 0
	for iter.Next() {
		count++
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (db *OpdsDB) Init() error {
	exists, err := db.Exists("nav", "root")
	if err != nil {
		return err
	}
	if !exists {
		err := db.Set("nav", "root", RootFeed)
		if err != nil {
			return err
		}
	}
	exists, err = db.Exists("nav", "all")
	if err != nil {
		return err
	}
	if !exists {
		err = db.Set("nav", "all", AllFeed)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *OpdsDB) GetFeed(name string) (*OpdsFeed, error) {
	dbFeed := &OpdsFeedDB{}
	err := db.Get("nav", name, dbFeed)
	if err != nil {
		return nil, err
	}
	feed := &OpdsFeed{OpdsCommon: dbFeed.OpdsCommon,
		XmlNs: "http://www.w3.org/2005/Atom"}
	if dbFeed.Type == Acq && dbFeed.All {
		n, err := db.Count("books")
		if err != nil {
			return nil, err
		}
		feed.Entries = make([]*OpdsEntry, n)
		iter, err := db.NewIterator("books")
		if err != nil {
			return nil, err
		}
		i := 0
		for iter.Next() {
			entry := &OpdsEntry{}
			err := json.Unmarshal(iter.Value(), entry)
			if err != nil {
				return nil, err
			}
			numLinks := 1
			if entry.Cover {
				numLinks++
			}
			if entry.Thumb {
				numLinks++
			}

			entry.Links = make([]*OpdsLink, numLinks)
			linkNo := 0
			entry.Links[linkNo] = &OpdsLink{Type: "application/epub+zip", Href: "/get/books/" + entry.Id, Rel: "http://opds-spec.org/acquisition"}
			linkNo++
			if entry.Cover {
				entry.Links[linkNo] = &OpdsLink{Type: entry.CoverType, Href: "/get/covers/" + entry.Id, Rel: "http://opds-spec.org/image"}
				linkNo++
			}
			if entry.Thumb {
				entry.Links[linkNo] = &OpdsLink{Type: entry.ThumbType, Href: "/get/thumbs/" + entry.Id, Rel: "http://opds-spec.org/image/thumbnail"}
				linkNo++
			}
			feed.Entries[i] = entry
			i++
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			return nil, err
		}
	}

	return feed, nil
}

func (db *OpdsDB) UpdateBook(uuid string, meta *OpdsMeta) error {
	entry := &OpdsEntry{}
	entry.OpdsMeta = meta
	entry.Id = uuid
	entry.Updated = time.Now().Format(time.RFC3339)
	err := db.Set("books", entry.Id, entry)
	if err != nil {
		return err
	}
	return  nil
}

func (db *OpdsDB) AddBook(meta *OpdsMeta) (string, error) {
	uuid := Uuidgen()
	return uuid,db.UpdateBook(uuid,meta)
}
