package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
)

// flags for the http server.
var serverFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "port",
		Value:  "8080",
		Usage:  "Port to run the server on",
		EnvVar: "PORT",
	},
	cli.StringFlag{
		Name:   "github.secret",
		Value:  "",
		Usage:  "Shared secret used by GitHub to sign webhook payloads. This secret will be used to verify that the request came from GitHub.",
		EnvVar: "GITHUB_SECRET",
	},
}

var cmdServer = cli.Command{
	Name:   "server",
	Usage:  "Run only the http server component.",
	Action: serverAction,
	Flags:  append(sharedFlags, serverFlags...),
}

func serverAction(c *cli.Context) {
	q := newBuildQueue(c)

	runServer(q, c)
}

func runServer(q conveyor.BuildQueue, c *cli.Context) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	port := c.String("port")
	info("Starting server on %s\n", port)

	errCh := make(chan error)
	go func() {
		errCh <- http.ListenAndServe(":"+port, newServer(q, c))
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		return nil
	}
}
