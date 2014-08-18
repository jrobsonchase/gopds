package gopds

import (
	"strings"
	"log"
	"encoding/json"
)

func scoreEntry(search string, entry *OpdsEntry) {
	search = strings.ToUpper(search)
	summary := strings.ToUpper(entry.Summary)
	title := strings.ToUpper(entry.Title)
	toSearch := []string{summary,title}
	if entry.Author != nil {
		author := strings.ToUpper(entry.Author.Name)
		toSearch = append(toSearch,author)
	}
	if entry.Content != nil {
		content := strings.ToUpper(entry.Content.Content)
		toSearch = append(toSearch,content)
	}
	words := strings.Split(search," ")
	numWords := len(words)
	for _,v := range toSearch {
		if strings.Contains(v,search) {
			entry.Order -= numWords
		}
		for _,w := range words {
			if strings.Contains(v,w) {
				entry.Order--
			}
		}
	}
}

func (srv *Server) Search(searchStr string) ([]*OpdsEntry,error) {
	db := srv.DB
	booksBytes,err := db.GetAll("books")
	if err != nil {
		return nil,err
	}
	books := make([]*OpdsEntry,len(booksBytes))
	for i,_ := range books {
		books[i] = &OpdsEntry{}
		err := json.Unmarshal(booksBytes[i],books[i])
		if err != nil {
			return nil,err
		}
		scoreEntry(searchStr,books[i])
	}
	filter := []*OpdsEntry{}
	for _,v := range books {
		if v.Order != 0 {
			createAcqLinks(v)
			filter = append(filter,v)
		}
	}
	log.Printf("Returning %d results",len(filter))
	return filter,nil
}
