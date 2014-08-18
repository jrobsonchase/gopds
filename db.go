package gopds

import (
	"sort"
	"encoding/json"
	"log"
	"time"
)

func (srv *Server) initDB() error {
	db := srv.DB
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

func (srv *Server) getFeedDB(name string,sortString string) (*OpdsFeed, error) {
	db := srv.DB
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
		feed.Entries,err = srv.getAcqEntries(dbFeed.Entries)
	case Nav:
		feed.Entries,err = srv.getNavEntries(dbFeed.Entries)
	default:
		feed.Entries,err = srv.Search(name[7:])
	}

	// add links to the feed
	createFeedLinks(feed,name)
	addSearchLink(feed)

	// Sort and paginate
	sorter := NewEntrySorter(feed.Entries,sortFun)
	sort.Sort(sorter)

	return feed, err
}

func (srv *Server) getAcqEntries(ents []string) ([]*OpdsEntry,error) {
	db := srv.DB
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

func (srv *Server) getNavEntries(ents []string) ([]*OpdsEntry,error) {
	db := srv.DB
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

func (srv *Server) updateBookDB(uuid string, meta *OpdsMeta) error {
	db := srv.DB
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

func (srv *Server) addBookDB(meta *OpdsMeta) (string, error) {
	uuid := Uuidgen()
	return uuid,srv.updateBookDB(uuid,meta)
}
