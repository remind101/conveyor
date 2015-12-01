package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"

	"code.google.com/p/go-uuid/uuid"
)

func main() {
	g := cloudwatch.NewGroup("test", cloudwatchlogs.New(defaults.DefaultConfig))

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
