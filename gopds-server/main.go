package main

import (
	"fmt"
	"sort"
	"net/http"
	"encoding/xml"
	"github.com/Pursuit92/gopds/epub"
	"github.com/Pursuit92/gopds"
)


func main() {
	srv,err := gopds.NewServer("database","files")
	if err != nil {
		panic(err)
	}

	srv.AutoAdd("autoadd",epub.ReadEpub)


	http.HandleFunc("/all",func(w http.ResponseWriter,r *http.Request) {
		feed,err := srv.GetFeed("all")
		if err != nil {
			panic(err)
		}
		sorter := gopds.NewEntrySorter(feed.Entries,gopds.SortTitle)
		sort.Sort(sorter)
		out,err := xml.MarshalIndent(feed,""," ")
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w,"%s\n%s\n",xml.Header,string(out))
	})
	http.Handle("/get/",http.StripPrefix("/get/", http.FileServer(http.Dir("files"))))
	http.ListenAndServe(":8000",nil)
}
