package gopds

import (
	"net/http"
	"strings"
	"log"
	"encoding/json"
	"fmt"
)

func stripPrefix(prefix string, fun http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := len(prefix)
		if r.URL.Path[:n] != prefix {
			http.Error(w,"Prefix not found",404)
			return
		}
		r.URL.Path = r.URL.Path[n:]
		fun(w,r)
	}
}

func handleFunc(pattern string, fun http.HandlerFunc) {
	l := len(pattern)
	if pattern[l-1] == '/' {
		http.HandleFunc(pattern,stripPrefix(pattern[:l-1],fun))
	} else {
		http.HandleFunc(pattern,fun)
	}
}

func (srv *Server) handleAPI(w http.ResponseWriter,r *http.Request) {
	components := strings.Split(r.URL.Path,"/")
	log.Printf("API Request:")
	log.Printf("%s, %d: %v",r.URL.Path,len(components),components)
	if len(components) > 1 {
		switch components[1] {
		case "book":
			stripPrefix("/book",srv.handleBook)(w,r)
		case "feed":
			stripPrefix("/feed",srv.handleFeed)(w,r)
		}
	}
}

func (srv *Server) handleBook(w http.ResponseWriter,r *http.Request) {
	components := strings.Split(r.URL.Path,"/")
	log.Printf("Book Request:")
	log.Printf("%s, %d: %v",r.URL.Path,len(components),components)
	if len(components) < 2 {
		http.Error(w,"Must give book uuid",404)
		return
	}
	id := components[1]
	switch r.Method {
	case "GET":
		book := &OpdsEntry{}
		err := srv.DB.Get("books",id,book)
		if err != nil {
			http.Error(w,"Book not found",404)
			return
		}
		out,_ := json.MarshalIndent(book.OpdsMeta,"","  ")
		fmt.Fprintf(w,"%s",out)
		return
	case "DELETE":
		err := srv.DelBook(id)
		if err != nil {
			http.Error(w,err.Error(),500)
		}
	case "PUT":
	}
}

func (srv *Server) handleFeed(w http.ResponseWriter,r *http.Request) {
	components := strings.Split(r.URL.Path,"/")
	log.Printf("Feed Request:")
	log.Printf("%s, %d: %v",r.URL.Path,len(components),components)
}
