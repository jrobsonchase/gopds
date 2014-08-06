package gopds

import (
	"errors"
	"github.com/howeyc/fsnotify"
	"path/filepath"
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

func (srv *Server) GetFeed(name string) (*OpdsFeed, error) {
	srv.Mut.Lock()
	defer srv.Mut.Unlock()
	return srv.DB.GetFeed(name)
}
