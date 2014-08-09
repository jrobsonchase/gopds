package gopds

import (
	"strings"
	"runtime/debug"
	"errors"
	"github.com/howeyc/fsnotify"
	"path/filepath"
	"encoding/xml"
	"net/http"
	"fmt"
	"time"
	"log"
	"io"
	"os"
	"sync"
)

type Server struct {
	DB    *OpdsDB
	Files string
	Mut *sync.Mutex
}

func NewServer(dataPath string) (*Server, error) {
	dbpath := filepath.FromSlash(dataPath + "/db")
	filePath := filepath.FromSlash(dataPath + "/files")
	db, err := OpenDB(dbpath)
	if err != nil {
		return nil, err
	}
	err = db.Init()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		for _, v := range []string{"books", "thumbs", "covers"} {
			err := os.MkdirAll(filepath.FromSlash(filePath+"/"+v), os.ModeDir|0777)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if !info.IsDir() {
			return nil, errors.New("Not a directory: " + filePath)
		}
	}
	return &Server{db, filePath,&sync.Mutex{}}, nil
}

func (srv *Server) AddBook(book Ebook) error {
	srv.Mut.Lock()
	defer srv.Mut.Unlock()
	defer book.Close()
	id, err := srv.DB.AddBook(book.OpdsMeta())
	if err != nil {
		return err
	}
	if book.OpdsMeta().Thumb {
		thumbPath := filepath.FromSlash(srv.Files + "/thumbs/" + id)
		file, err := os.Create(thumbPath)
		if err != nil {
			return err
		}
		defer file.Close()
		thumb := book.Thumb()
		defer thumb.Close()
		_, err = io.Copy(file, thumb)
		if err != nil {
			return err
		}
	}
	if book.OpdsMeta().Cover {
		coverPath := filepath.FromSlash(srv.Files + "/covers/" + id)
		file, err := os.Create(coverPath)
		if err != nil {
			return err
		}
		defer file.Close()
		cover := book.Cover()
		defer cover.Close()
		_, err = io.Copy(file, cover)
		if err != nil {
			return err
		}
	}
	bookPath := filepath.FromSlash(srv.Files + "/books/" + id)
	file, err := os.Create(bookPath)
	if err != nil {
		return err
	}
	defer file.Close()
	bookFile := book.Book()
	defer bookFile.Close()
	_, err = io.Copy(file, bookFile)
	if err != nil {
		return err
	}
	return nil
}

func (srv *Server) AutoAdd(filePath string, open func(string) (Ebook,error)) error {
	safePath := filepath.FromSlash(filePath)
	watch,err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
	safePathInfo, err := os.Stat(safePath)
	if err != nil {
		err := os.MkdirAll(safePath, os.ModeDir|0777)
		if err != nil {
			return  err
		}
	} else {
		if !safePathInfo.IsDir() {
			return errors.New("Not a directory")
		}
	}
    err = watch.Watch(safePath)
    if err != nil {
        return err
    }

    go func() {
        for {
            select {
            case ev := <-watch.Event:
                if ev.IsCreate() {
                    fileName := ev.Name
                    if fileName[len(fileName)-4:] == "epub" {
                        go func(name string) {
                            <-time.After(1 * time.Second)
                            book,err := open(name)
                            if err == nil {
                                log.Printf("Adding %s",name)
                                srv.AddBook(book)
                                os.Remove(name)
                            } else {
                                log.Printf("Error: %s",err.Error())
                            }
                        }(fileName)
                    }
                }
            }
        }
    }()
	return nil
}

func (srv *Server) GetFeed(name string,sortOverride string) (string, error) {
	srv.Mut.Lock()
	feed,err := srv.DB.GetFeed(name,sortOverride)
	srv.Mut.Unlock()
	if err != nil {
		return "",err
	}
	out,err := xml.MarshalIndent(feed,"","  ")
	return string(out),err
}

func (srv *Server) GetBookFeed(id string) (string,error) {
	srv.Mut.Lock()
	feed,err := srv.DB.GetBookFeed(id)
	srv.Mut.Unlock()
	if err != nil {
		return "",err
	}
	out,err := xml.MarshalIndent(feed,"","  ")
	return string(out),err
}


func (srv *Server) serveFeed(feed,sortMeth string) func(w http.ResponseWriter,r *http.Request) {
	return func(w http.ResponseWriter,r *http.Request) {
		var err error
		feed,err := srv.GetFeed(feed,sortMeth)
		if err != nil {
			http.Error(w,err.Error(),500)
			return
		}
        fmt.Fprintf(w,"%s\n%s\n",xml.Header,feed)
	}
}

func (srv *Server) handleCatalog(w http.ResponseWriter,r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(w,"%s\n%s",r,debug.Stack())
		}
	}()
	path := r.URL.Path
	components := strings.Split(path,"/")
	log.Printf("Path: %s",path)
	log.Printf("Components: %d %v",len(components),components)
	var feed,sortMeth string
	if  components[1] != "" {
		feed = components[1]
	} else {
		feed = "root"
	}
	if len(components) > 3 && components[2] == "sort" {
		sortMeth = components[3]
		log.Print("Sorting by",sortMeth)
	}
	srv.serveFeed(feed,sortMeth)(w,r)
}

func (srv *Server) handleSearch(w http.ResponseWriter,r *http.Request) {
	searchTerms := r.FormValue("q")
	log.Print("Searching: " + searchTerms)
	srv.serveFeed("search:"+searchTerms,"")(w,r)
}

func (srv *Server) handleBook(w http.ResponseWriter,r *http.Request) {
	id := r.FormValue("id")
	log.Print("Serving book: "+id)
	if id == "" {
		srv.serveFeed("all","")(w,r)
	} else {
		srv.serveBook(id)(w,r)
	}
}

func (srv *Server) serveBook(id string) func(http.ResponseWriter,*http.Request) {
	return func(w http.ResponseWriter,r *http.Request) {
		var err error
		feed,err := srv.GetBookFeed(id)
		if err != nil {
			http.Error(w,err.Error(),500)
			return
		}
        fmt.Fprintf(w,"%s\n%s\n",xml.Header,feed)
	}

}

func (srv *Server) ServeHTTP() error {
    http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request) {
        http.Redirect(w,r,"/catalog",301)
    })

	http.HandleFunc("/search",srv.handleSearch)
	http.HandleFunc("/book",srv.handleBook)
    http.Handle("/catalog/",http.StripPrefix("/catalog",http.HandlerFunc(srv.handleCatalog)))
    http.Handle("/get/",http.StripPrefix("/get/", http.FileServer(http.Dir(srv.Files))))
    return http.ListenAndServe(":8080",nil)
}
