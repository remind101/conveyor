package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/remind101/conveyor/pkg/cloudwatch"
)

func main() {
	sess := session.Must(session.NewSession())

	g := cloudwatch.NewGroup("test", cloudwatchlogs.New(sess))

	stream := uuid.New()

	w, err := g.Create(stream)
	if err != nil {
		log.Fatal(err)
	}

	r, err := g.Open(stream)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		var i int
		for {
			i++
			<-time.After(time.Second / 30)
			_, err := fmt.Fprintf(w, "Line %d\n", i)
			if err != nil {
				log.Println(err)
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 10)
		log.Println("Time to shutdown logsteam")
		r.Close()
	}()

	_, err = io.Copy(os.Stdout, r)

	log.Println("We are at the end of this")

}
