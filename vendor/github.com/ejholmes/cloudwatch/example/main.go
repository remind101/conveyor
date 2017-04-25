package main

import (
	"io"
	"log"
	"os"
	"time"
	"fmt"
	
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
	"github.com/pborman/uuid"
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

	io.Copy(os.Stdout, r)
}
