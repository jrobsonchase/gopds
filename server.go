package gopds

import (
	"errors"
	"github.com/howeyc/fsnotify"
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

func NewServer(dbpath, filepath string) (*Server, error) {
	db, err := OpenDB(dbpath)
	if err != nil {
		return nil, err
	}
	err = db.Init()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filepath)
	if err != nil {
		for _, v := range []string{"books", "thumbs", "covers"} {
			err := os.MkdirAll(filepath+"/"+v, os.ModeDir|0777)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if !info.IsDir() {
			return nil, errors.New("Not a directory: " + filepath)
		}
	}
	return &Server{db, filepath,&sync.Mutex{}}, nil
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
		file, err := os.Create(srv.Files + "/thumbs/" + id)
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
		file, err := os.Create(srv.Files + "/covers/" + id)
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
	file, err := os.Create(srv.Files + "/books/" + id)
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

func (srv *Server) AutoAdd(path string, open func(string) (Ebook,error)) error {
	watch,err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    err = watch.Watch(path)
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
                                os.Rename(name,"error/" + name)
                            }
                        }(fileName)
                    }
                }
            }
        }
    }()
	return nil
}

func (srv *Server) GetFeed(name string) (*OpdsFeed, error) {
	srv.Mut.Lock()
	defer srv.Mut.Unlock()
	return srv.DB.GetFeed(name)
}
