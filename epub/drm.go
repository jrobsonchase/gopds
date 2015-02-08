package epub

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Pursuit92/gopds"
)

var (
	keyStore string = ""
	ignoble  string = "/usr/local/bin/ignobleepub.py"
	deDRM    bool   = false
)

func init() {
	flag.BoolVar(&deDRM, "dedrm", false, "Toggle DRM Removal")
}

func removeDRM(path string) string {
	log.Print("Removing drm...")
	keys, _ := filepath.Glob(filepath.FromSlash(keyStore + "/*.b64"))
	for _, v := range keys {
		cmd := exec.Command(ignoble, v, path, path+"-dedrm")
		err := cmd.Run()
		if err != nil {
			if err.Error() == "exit status 1" {
				log.Print("Book is drm free!")
				return path
			}
		} else {
			log.Print("Success!")
			return path + "-dedrm"
		}
	}
	return ""

}

func RemoveDRM(path string) (gopds.Ebook, error) {
	out := removeDRM(path)
	return nil, errors.New("Removed DRM: " + out)
}

func AddKey(storage string) func(string) (gopds.Ebook, error) {
	safe := filepath.FromSlash(storage)
	safeInfo, err := os.Stat(safe)
	if err != nil {
		err := os.MkdirAll(safe, os.ModeDir|0777)
		if err != nil {
			return func(path string) (gopds.Ebook, error) {
				return nil, err
			}
		}
	} else {
		if !safeInfo.IsDir() {
			return func(path string) (gopds.Ebook, error) {
				return nil, errors.New("Key storage isn't a directory!")
			}
		}
	}
	keyStore = safe
	return func(path string) (gopds.Ebook, error) {
		err := os.Rename(path, filepath.FromSlash(safe+"/"+gopds.Uuidgen()+".b64"))
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Imported key file")
	}
}
