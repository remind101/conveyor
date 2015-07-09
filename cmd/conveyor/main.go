package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/remind101/conveyor"
)

func main() {
	var port = flag.String("port", "8080", "The port to bind to.")
	s, err := conveyor.NewServerFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on " + *port)
	log.Fatal(http.ListenAndServe(":"+*port, s))
}
