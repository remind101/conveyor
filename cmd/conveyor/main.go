package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
)

// flags shared between the server and worker subcommands.
var sharedFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "queue",
		Value:  "memory://",
		Usage:  "Build queue to use. Defaults to an in memory queue.",
		EnvVar: "QUEUE",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "conveyor"
	app.Usage = "Build docker images from GitHub repositories"
	app.Action = mainAction
	app.Flags = append(
		sharedFlags,
		append(workerFlags, serverFlags...)...,
	)
	app.Commands = []cli.Command{
		cmdServer,
		cmdWorker,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func mainAction(c *cli.Context) {
	q := newBuildQueue(c)

	worker := make(chan error)
	server := make(chan error)

	go func() {
		worker <- runWorker(q, c)
	}()

	go func() {
		server <- runServer(q, c)
	}()

	<-worker
	<-server
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

var info = fmt.Printf
