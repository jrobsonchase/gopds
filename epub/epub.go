package epub

import (
	"archive/zip"
	"path/filepath"
	"regexp"
	"encoding/xml"
	"github.com/Pursuit92/gopds"
	"io"
	"os"
)

var (
	coverIds []string = []string{"fcvi", "coverimagestandard"}
	thumbIds []string = []string{"fcvt", "thumbimagestandard"}
	coverExp *regexp.Regexp = regexp.MustCompile(`.*([c,C]over|cvi).*\.(jpg|jpeg|png)`)
	thumbExp *regexp.Regexp = regexp.MustCompile(`.*cvt.*\.(jpg|jpeg|png)`)
)

type Epub struct {
	path string
	file *zip.ReadCloser
	*Package
	HasCover, HasThumb bool
	ThumbType, CoverType string
}

func ReadEpub(path string) (gopds.Ebook, error) {
	return readEpub(path)
}

func readEpub(path string) (*Epub, error) {
	safePath := filepath.FromSlash(path)

	book := &Epub{path: safePath, HasThumb: true, HasCover: true}
	// Open a zip archive for reading.
	var err error
	book.file, err = zip.OpenReader(safePath)
	if err != nil {
		return nil,err
	}
	err = book.readOPF()
	if err != nil {
		return nil, err
	}
	book.CoverType,book.HasCover = book.coverTest()
	book.ThumbType,book.HasThumb = book.thumbTest()
	return book, nil
}

func (book Epub) coverTest() (string,bool) {
	for _, v := range book.file.File {
		if coverExp.Match([]byte(v.Name)) {
			_,err := v.Open()
			if err == nil {
				imgtype := ""
				switch v.Name[len(v.Name)-4:] {
				case ".jpg","jpeg":
					imgtype = "image/jpeg"
				case ".png":
					imgtype = "image/png"
				}
				return imgtype,true
			}
		}
	}
	return "",false
}

func (book Epub) thumbTest() (string,bool) {
	for _, v := range book.file.File {
		if thumbExp.Match([]byte(v.Name)) {
			_,err := v.Open()
			if err == nil {
				imgtype := ""
				switch v.Name[len(v.Name)-4:] {
				case ".jpg",".jpeg":
					imgtype = "image/jpeg"
				case ".png":
					imgtype = "image/png"
				}
				return imgtype,true
			}
		}
	}
	return "",false
}

func (book Epub) Thumb() io.ReadCloser {
	for _, v := range book.file.File {
		if thumbExp.Match([]byte(v.Name)) {
			file,_ := v.Open()
			return file
		}
	}
	return nil
}

func (book Epub) Cover() io.ReadCloser {
	for _, v := range book.file.File {
		if coverExp.Match([]byte(v.Name)) {
			file,_ := v.Open()
			return file
		}
	}
	return nil
}

func (book Epub) Book() io.ReadCloser {
	file, _ := os.Open(book.path)
	return file
}

func (book *Epub) readOPF() error {
	// Iterate through the files in the archive,
	for _, f := range book.file.File {
		if f.Name[len(f.Name)-3:] == "opf" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			opf := &Package{}
			err = xml.NewDecoder(rc).Decode(opf)
			if err != nil {
				return err
			}
			book.Package = opf
			return nil
		}
	}
	return nil
}

func (book Epub) OpdsMeta() *gopds.OpdsMeta {
	meta := book.Meta
	return &gopds.OpdsMeta{Title: meta.Title,
	Author:    &gopds.OpdsAuthor{Name: meta.Creator},
	Publisher: meta.Publisher,
	Issued:    meta.Date,
	Lang:      meta.Language,
	Summary:   meta.Description,
	Rights:    meta.Rights,
	Cover:     book.HasCover,
	Thumb:     book.HasThumb,
	CoverType: book.CoverType,
	ThumbType: book.ThumbType}
}

func (book *Epub) Close() {
	book.file.Close()
}
