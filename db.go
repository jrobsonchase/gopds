package gopds

import (
	"encoding/json"
	"fmt"
	"sort"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"os"
	"time"
	"path/filepath"
)

type OpdsDB struct {
	path string
	dbs  map[string]*leveldb.DB
}

func OpenDB(path string) (*OpdsDB, error) {
	safePath := filepath.FromSlash(path)
	database := &OpdsDB{path: safePath}
	database.dbs = make(map[string]*leveldb.DB)
	safePathInfo, err := os.Stat(safePath)
	if err != nil {
		err := os.MkdirAll(safePath, os.ModeDir|0777)
		if err != nil {
			return nil, err
		}
	} else {
		if !safePathInfo.IsDir() {
			return nil, errors.New("Not a directory")
		}
	}
	return database, nil
}

func (db *OpdsDB) GetDB(database string) (*leveldb.DB, error) {
	_, exists := db.dbs[database]
	if !exists {
		newdb, err := leveldb.OpenFile(filepath.FromSlash(db.path+"/"+database), nil)
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

func (db *OpdsDB) GetFeed(name string,perPage,pageNo int,sortString string) (*OpdsFeed, error) {
	dbFeed := &OpdsFeedDB{}
	err := db.Get("nav", name, dbFeed)
	if err != nil {
		return nil, err
	}

	feed := &OpdsFeed{OpdsCommon: dbFeed.OpdsCommon,
	XmlNs: "http://www.w3.org/2005/Atom"}

	sortFun := sortFuncBytes[dbFeed.Sort]
	if sortString != "" {
		sortFun = sortFuncStrings[sortString]
	}

	switch dbFeed.Type {
	case Acq:
		feed.Entries,err = db.getAcqEntries(dbFeed.Entries)
	case Nav:
		feed.Entries,err = db.getNavEntries(dbFeed.Entries)
	default:
		return nil,errors.New("Not yet implemented")
	}

	// add links to the feed
	createFeedLinks(feed,name,perPage,pageNo,sortString)
	addSearchLink(feed)

	// Sort and paginate
	sorter := NewEntrySorter(feed.Entries,sortFun)
	sort.Sort(sorter)
	start := perPage * pageNo
	end := start + perPage
	if end > len(feed.Entries) || perPage == 0 {
		end = len(feed.Entries)
	}
	if start > len(feed.Entries) {
		start = len(feed.Entries)
	}
	feed.Entries = feed.Entries[start:end]

	return feed, err
}

func (db *OpdsDB) getAcqEntries(ents []string) ([]*OpdsEntry,error) {
	var entries []*OpdsEntry
	if ents == nil {
		n, err := db.Count("books")
		if err != nil {
			return nil, err
		}
		entries = make([]*OpdsEntry, n)
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
			createAcqLinks(entry)
			entry.Id = "urn:uuid:" + entry.Id
			entries[i] = entry
			i++
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			return nil, err
		}
	} else {
		entries = make([]*OpdsEntry,len(ents))
		n := 0
		for i,v := range ents {
			err := db.Get("books",v,entries[i])
			if err == nil {
				createAcqLinks(entries[i])
				entries[i].Id = "urn:uuid:" + entries[i].Id
				n++
			}
		}
		if n != len(ents) {
			realEntries := make([]*OpdsEntry,n)
			j := 0
			for _,v := range entries {
				if v != nil {
					realEntries[j] = v
					j++
				}
			}
			entries = realEntries
		}
	}

	return entries,nil
}

func feedToEntry(f *OpdsFeedDB) *OpdsEntry {
	entry := &OpdsEntry{}
	entry.Id = f.Id
	entry.Title = f.Title
	entry.Content = &OpdsContent{Content: f.Desc}
	return entry
}

func (db *OpdsDB) getNavEntries(ents []string) ([]*OpdsEntry,error) {
	var entries []*OpdsEntry
	if ents == nil {
		n, err := db.Count("nav")
		if err != nil {
			return nil, err
		}
		entries = make([]*OpdsEntry, n)
		iter, err := db.NewIterator("nav")
		if err != nil {
			return nil, err
		}
		i := 0
		for iter.Next() {
			entry := &OpdsFeedDB{}
			err := json.Unmarshal(iter.Value(), entry)
			if err != nil {
				return nil, err
			}
			entry.Id = "urn:uuid:" + entry.Id
			entries[i] = feedToEntry(entry)
			createNavLinks(entries[i])
			i++
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			return nil, err
		}
	} else {
		dbEntries := make([]*OpdsFeedDB,len(ents))
		n := 0
		for i,v := range ents {
			err := db.Get("nav",v,dbEntries[i])
			if err == nil {
				dbEntries[i].Id = "urn:uuid:" + dbEntries[i].Id
				n++
			}
		}
		if n != len(ents) {
			realEntries := make([]*OpdsEntry,n)
			j := 0
			for _,v := range dbEntries {
				if v != nil {
					realEntries[j] = feedToEntry(v)
					createNavLinks(realEntries[j])
					j++
				}
			}
			entries = realEntries
		}
	}

	return entries,nil
}

func createNavLinks(feed *OpdsEntry) {
}

func addSearchLink(feed *OpdsFeed) {
	// <link  rel="search" title="Search Catalog" type="application/atom+xml" href="http://manybooks.net/opds/search.php?q={searchTerms}"/>
	searchLink := &OpdsLink{Rel: "search",
		Href: "/catalog/search?q={searchTerms}",
		Type: "application/atom+xml"}
	if feed.Links != nil {
		feed.Links = append(feed.Links,searchLink)
	} else {
		feed.Links = []*OpdsLink{searchLink}
	}
}

func createFeedLinks(feed *OpdsFeed,name string,perPage,pageNo int, sorter string) {
	var countStr,pageStr,prevStr,nextStr,endStr,sortStr string
	if perPage != 0 {
		countStr = fmt.Sprintf(";count=%d",perPage)
		pageStr = fmt.Sprintf(";page=%d",pageNo)
		prevStr = fmt.Sprintf(";page=%d",pageNo-1)
		nextStr = fmt.Sprintf(";page=%d",pageNo+1)
		endStr = fmt.Sprintf(";page=%d",len(feed.Entries) / perPage)
	}
	if sorter != "" {
		sortStr = "sort="+sorter
	}
	base := "/catalog/"+name+"?"+sortStr+countStr
	numLinks := 3
	navType := "application/atom+xml;profile=opds-catalog;kind=navigation"
	start := base + ";page=0"
	end := ""
	if perPage != 0 {
		end = base + endStr
	} else {
		end = start
	}
	self := base + pageStr
	prev := ""
	next := ""
	if perPage != 0 {
		if pageNo != 0 {
			prev = base + prevStr
			numLinks++
		}
		if pageNo * perPage < len(feed.Entries) {
			next = base + nextStr
			numLinks++
		}

	}
	newLinks := make([]*OpdsLink,numLinks)
	linkNo := 3
	newLinks[0] = &OpdsLink{Type: navType,Href:self,Rel: "self"}
	newLinks[1] = &OpdsLink{Type: navType, Href:start, Rel: "start"}
	newLinks[2] = &OpdsLink{Type: navType, Href:end, Rel: "end"}
	if prev != "" {
		newLinks[3] = &OpdsLink{Href:prev, Rel: "prev"}
		linkNo++
	}
	if next != "" {
		newLinks[linkNo] = &OpdsLink{Href:next, Rel: "next"}
		linkNo++
	}

	if feed.Links != nil {
		feed.Links = append(feed.Links,newLinks...)
	} else {
		feed.Links = newLinks
	}

}

func createAcqLinks(entry *OpdsEntry) {
	numLinks := 1
	if entry.Cover {
		numLinks++
	}
	if entry.Thumb {
		numLinks++
	}

	entry.Links = make([]*OpdsLink, numLinks)
	linkNo := 0
	entry.Links[linkNo] = &OpdsLink{Type: "application/epub+zip",
	Href: "/get/books/" + entry.Id,
	Rel: "http://opds-spec.org/acquisition"}
	linkNo++
	if entry.Cover {
		entry.Links[linkNo] = &OpdsLink{Type: entry.CoverType,
		Href: "/get/covers/" + entry.Id,
		Rel: "http://opds-spec.org/image"}
		linkNo++
	}
	if entry.Thumb {
		entry.Links[linkNo] = &OpdsLink{Type: entry.ThumbType,
		Href: "/get/thumbs/" + entry.Id,
		Rel: "http://opds-spec.org/image/thumbnail"}
		linkNo++
	}
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
