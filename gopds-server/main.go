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
	port := flag.Int("port",8080,"Listen port")
	flag.Parse()

	srv,err := gopds.NewServer(*dataPath,*autoadd)
	if err != nil {
		panic(err)
	}

	if *autoadd != "" {
		srv.AutoAdd("epub",epub.ReadEpub)
		srv.AutoAdd("b64",epub.AddKey("keystorage"))
	}


	log.Fatal(srv.ServeHTTP(*port))
}
