package main

import (
	"flag"
	"log"
	"github.com/Pursuit92/gopds/epub"
	"github.com/Pursuit92/gopds"
)


func main() {
	autoadd := flag.String("autoadd","","Directory to watch for epubs")
	dataPath := flag.String("data",".gopds","Data directory")
	flag.Parse()

	srv,err := gopds.NewServer(*dataPath)
	if err != nil {
		panic(err)
	}

	if *autoadd != "" {
		srv.AutoAdd(*autoadd,epub.ReadEpub)
	}


	log.Fatal(srv.ServeHTTP())
}
