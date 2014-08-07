package gopds

import (
	"errors"
	"strconv"
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

func (srv *Server) GetFeed(name string,perPage,pageNo int,sortOverride string) (string, error) {
	srv.Mut.Lock()
	feed,err := srv.DB.GetFeed(name,perPage,pageNo,sortOverride)
	srv.Mut.Unlock()
	if err != nil {
		return "",err
	}
	out,err := xml.MarshalIndent(feed,"","  ")
	return string(out),err
}

func (srv *Server) handleFeed(feed string) func(w http.ResponseWriter,r *http.Request) {
	return func(w http.ResponseWriter,r *http.Request) {
		var perPage,pageNo int
		var err error
		count := r.FormValue("count")
		if count != "" {
			perPage,err = strconv.Atoi(count)
			if err != nil {
				http.Error(w,err.Error(),400)
				return
			}
		}
		page := r.FormValue("page")
		if page != "" {
			pageNo,err = strconv.Atoi(page)
			if err != nil {
				http.Error(w,err.Error(),400)
				return
			}
		}
		sortMeth := r.FormValue("sort")
		feed,err := srv.GetFeed(feed,perPage,pageNo,sortMeth)
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

    http.HandleFunc("/catalog",srv.handleFeed("all"))
    http.Handle("/get/",http.StripPrefix("/get/", http.FileServer(http.Dir(srv.Files))))
    return http.ListenAndServe(":8080",nil)
}
