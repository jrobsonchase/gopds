package gopds

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
