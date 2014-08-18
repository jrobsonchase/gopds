package gopds

import (
	"strings"
	//"encoding/json"
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
	opdsdb "github.com/Pursuit92/gopds/db"
)

type Server struct {
	DB    *opdsdb.OpdsDB
	Files string
	AutoAddPath string
	addPatterns []AddPattern
	Mut *sync.Mutex
}

func NewServer(dataPath,addPath string) (*Server, error) {
	dbpath := filepath.FromSlash(dataPath + "/db")
	filePath := filepath.FromSlash(dataPath + "/files")
	db, err := opdsdb.OpenDB(dbpath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		for _, v := range []string{"books", "thumbs", "covers", "tmp"} {
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
	srv := &Server{db, filePath,addPath,[]AddPattern{},&sync.Mutex{}}
	err = srv.initDB()
	if err != nil {
		return nil, err
	}
	err = srv.runAutoAdds()
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (srv *Server) AddBook(book Ebook) error {
	srv.Mut.Lock()
	defer srv.Mut.Unlock()
	defer book.Close()
	id, err := srv.addBookDB(book.OpdsMeta())
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

func (srv *Server) DelBook(id string) error {
	book := &OpdsEntry{}
	err := srv.DB.Get("books",id,book)
	if err != nil {
		return err
	}
	if book.Cover {
		os.Remove(filepath.FromSlash(srv.Files + "/covers/" + id))
	}
	if book.Thumb {
		os.Remove(filepath.FromSlash(srv.Files + "/thumbs/" + id))
	}
	os.Remove(filepath.FromSlash(srv.Files + "/books/" + id))
	return srv.DB.Del("books",id)
}

type AddPattern struct {
	Pattern string
	Open func(string) (Ebook, error)
}

func (srv *Server) AutoAdd(extension string, open func(string) (Ebook,error)) {
	srv.addPatterns = append(srv.addPatterns,AddPattern{extension,open})
}

func (srv *Server) runAutoAdds() error {
	filePath := srv.AutoAddPath
	if filePath == "" {
		return nil
	}
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
					for _,v := range srv.addPatterns {
						l := len(v.Pattern)
						if fileName[len(fileName)-l:] == v.Pattern {
							go func(name string,open func(string) (Ebook,error)) {
								<-time.After(1 * time.Second)
								book,err := open(name)
								if err == nil {
									log.Printf("Adding %s",name)
									srv.AddBook(book)
									os.Remove(name)
								} else {
									log.Printf("Error: %s",err.Error())
								}
							}(fileName,v.Open)
						}
					}
				}
			}
		}
	}()
	return nil
}

func xmlMarshaler(i interface{},s1,s2 string) ([]byte,error) {
	marshalled,err := xml.MarshalIndent(i,s1,s2)
	if err != nil {
		return marshalled,err
	}
	out := fmt.Sprintf("%s\n%s\n",xml.Header,marshalled)
	return []byte(out),nil
}

func (srv *Server) GetFeed(name string,sortOverride string,marsh func(interface{},string,string) ([]byte,error)) (string, error) {
	srv.Mut.Lock()
	feed,err := srv.getFeedDB(name,sortOverride)
	srv.Mut.Unlock()
	if err != nil {
		return "",err
	}
	out,err := marsh(feed,"","  ")
	return string(out),err
}

func (srv *Server) serveFeed(feed,sortMeth string) func(w http.ResponseWriter,r *http.Request) {
	return func(w http.ResponseWriter,r *http.Request) {
		var err error
		feed,err := srv.GetFeed(feed,sortMeth,xmlMarshaler)
		if err != nil {
			http.Error(w,err.Error(),500)
			return
		}
        fmt.Fprintf(w,"%s\n",feed)
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

func (srv *Server) ServeHTTP(port int) error {
    http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request) {
		log.Printf("Got %s, redirecting...",r.URL.Path)
        http.Redirect(w,r,"/catalog",301)
    })
	handleFunc("/api/",srv.handleAPI)
	handleFunc("/search",srv.handleSearch)
    handleFunc("/catalog/",srv.handleCatalog)
    http.Handle("/get/",http.StripPrefix("/get/", http.FileServer(http.Dir(srv.Files))))
    return http.ListenAndServe(":"+fmt.Sprintf("%d",port),nil)
}
