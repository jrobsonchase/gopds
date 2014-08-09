package gopds

import (
	"encoding/json"
	"log"
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

func (db *OpdsDB) GetAll(database string) ([][]byte,error) {
	num,err := db.Count(database)
	if err != nil {
		return nil,err
	}
	iter,err := db.NewIterator(database)
	if err != nil {
		return nil,err
	}
	out := make([][]byte, num)
	i := 0
	for iter.Next() {
		out[i] = make([]byte,len(iter.Value()))
		copy(out[i],iter.Value())
		i++
	}
	iter.Release()
	return out,iter.Error()
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

func (db *OpdsDB) GetBookFeed(id string) (*OpdsFeed, error) {
	entries,err := db.getAcqEntries([]string{id})
	if err != nil {
		return nil,err
	}
	if len(entries) != 0 {
		feed := &OpdsFeed{OpdsCommon: &OpdsCommon{
			Title: entries[0].Title,
			Id: "urn:uuid:" + Uuidgen(),
			Type: Acq},
			XmlNs: "http://www.w3.org/2005/Atom"}
		feed.Entries = entries

		createFeedLinks(feed,"book:"+id)
		addSearchLink(feed)
		return feed,nil
	}
	log.Print("No book found by id: "+id)
	return nil,nil
}

func (db *OpdsDB) GetFeed(name string,sortString string) (*OpdsFeed, error) {
	var err error
	dbFeed := &OpdsFeedDB{}
	if len(name) >= 7 && name[:7] == "search:" {
		dbFeed = &OpdsFeedDB{OpdsCommon: &OpdsCommon{
			Id: "urn:uuid:" + Uuidgen(),
			Type: Search,
			Title: "Search Results"},
			Desc: "Search: " + name[7:],
			Sort: SortOrder}
	} else {
		err = db.Get("nav", name, dbFeed)
		if err != nil {
			dbFeed = &OpdsFeedDB{OpdsCommon: &OpdsCommon{
				Id: "urn:uuid:" + Uuidgen(),
				Type: Search,
				Title: "Feed not found, searching: " + name},
				Desc: "Search: " + name,
				Sort: SortOrder}
		}
	}

	feed := &OpdsFeed{OpdsCommon: dbFeed.OpdsCommon,
	XmlNs: "http://www.w3.org/2005/Atom"}

	sortFun := sortFuncBytes[dbFeed.Sort]
	if sortString != "" && dbFeed.Type != Search {
		sortFun = sortFuncStrings[sortString]
	}

	switch dbFeed.Type {
	case Acq:
		feed.Entries,err = db.getAcqEntries(dbFeed.Entries)
	case Nav:
		feed.Entries,err = db.getNavEntries(dbFeed.Entries)
	default:
		feed.Entries,err = db.Search(name[7:])
	}

	// add links to the feed
	createFeedLinks(feed,name)
	addSearchLink(feed)

	// Sort and paginate
	sorter := NewEntrySorter(feed.Entries,sortFun)
	sort.Sort(sorter)

	return feed, err
}

func (db *OpdsDB) getAcqEntries(ents []string) ([]*OpdsEntry,error) {
	var entries []*OpdsEntry
	if ents == nil {
		entriesBytes,err := db.GetAll("books")
		if err != nil {
			return nil,err
		}
		entries = make([]*OpdsEntry,len(entriesBytes))
		for i,_ := range entries {
			entries[i] = &OpdsEntry{}
			err := json.Unmarshal(entriesBytes[i],entries[i])
			if err != nil {
				return nil,err
			}
		}
	} else {
		entries = make([]*OpdsEntry,len(ents))
		n := 0
		for i,v := range ents {
			entries[i] = &OpdsEntry{}
			err := db.Get("books",v,entries[i])
			if err == nil {
				n++
			} else {
				log.Print("Error: "+err.Error())
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

	for _,v := range entries {
		createAcqLinks(v)
		v.Id = "urn:uuid:" + v.Id
	}

	return entries,nil
}

func feedToEntry(f *OpdsFeedDB) *OpdsEntry {
	entry := &OpdsEntry{OpdsMeta: &OpdsMeta{}}
	entry.Id = f.Id
	entry.Updated = f.Updated
	entry.Author = f.Author
	entry.Title = f.Title
	entry.Category = f.Name
	entry.Content = &OpdsContent{Content: f.Desc}
	return entry
}

func (db *OpdsDB) getNavEntries(ents []string) ([]*OpdsEntry,error) {
	var entries []*OpdsEntry
	var feeds []*OpdsFeedDB
	if ents == nil {
		feedsBytes,err := db.GetAll("nav")
		if err != nil {
			return nil,err
		}
		feeds = make([]*OpdsFeedDB,len(feedsBytes))
		for i,_ := range feeds {
			feeds[i] = &OpdsFeedDB{}
			err := json.Unmarshal(feedsBytes[i],feeds[i])
			if err != nil {
				return nil,err
			}
		}
	} else {
		feeds = make([]*OpdsFeedDB,len(ents))
		n := 0
		for i,v := range ents {
			feeds[i] = &OpdsFeedDB{}
			err := db.Get("nav",v,feeds[i])
			if err == nil {
				n++
			} else {
			}
		}
		if n != len(ents) {
			realFeeds := make([]*OpdsFeedDB,n)
			j := 0
			for _,v := range feeds {
				if v != nil {
					realFeeds[j] = v
					j++
				}
			}
			feeds = realFeeds
		}
	}

	entries = make([]*OpdsEntry,len(feeds))
	for i,v := range feeds {
		entries[i] = feedToEntry(v)
		createNavLinks(entries[i])
		entries[i].Id = "urn:uuid:" + entries[i].Id
	}

	return entries,nil
}

func createNavLinks(feed *OpdsEntry) {
	// <link type="application/atom+xml" href="http://manybooks.net/opds/new_titles.php"/>
	link := &OpdsLink{Href: "/catalog/" + feed.Category,
		Type: "application/atom+xml"}
	if feed.Links == nil {
		feed.Links = []*OpdsLink{link}
	} else {
		feed.Links = append(feed.Links,link)
	}
}

func addSearchLink(feed *OpdsFeed) {
	searchLink := &OpdsLink{Rel: "search",
		Href: "/search?q={searchTerms}",
		Type: "application/atom+xml"}
	if feed.Links != nil {
		feed.Links = append(feed.Links,searchLink)
	} else {
		feed.Links = []*OpdsLink{searchLink}
	}
}

func createFeedLinks(feed *OpdsFeed,name string) {
	var feedType string
	switch feed.Type {
	case Nav:
		feedType = "application/atom+xml;profile=opds-catalog;kind=navigation"
	case Acq:
		feedType = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	default:
		feedType = "application/atom+xml"
	}
	var base string
	if len(name) >= 7 && name[:7] == "search:" {
		base = "/search?q="+name[7:]
	} else if len(name) >= 5 && name[:5] == "book:" {
		base = "/book?id="+name[5:]
	} else {
		base = "/catalog/"+name
	}
	selfLink := &OpdsLink{Type: feedType,Href:base,Rel: "self"}
	newLinks := []*OpdsLink{selfLink}

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
