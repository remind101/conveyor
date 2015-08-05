package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
)

var cmdServer = cli.Command{
	Name:   "server",
	Usage:  "Run an http server to build Docker images whenever a push event happens on GitHub",
	Action: runServer,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "port",
			Value:  "8080",
			Usage:  "Port to run the server on",
			EnvVar: "PORT",
		},
		cli.StringFlag{
			Name:   "github.token",
			Value:  "",
			Usage:  "GitHub API token to use when updating commit statuses on repositories.",
			EnvVar: "GITHUB_TOKEN",
		},
		cli.StringFlag{
			Name:   "github.secret",
			Value:  "",
			Usage:  "Shared secret used by GitHub to sign webhook payloads. This secret will be used to verify that the request came from GitHub.",
			EnvVar: "GITHUB_SECRET",
		},
		cli.BoolFlag{
			Name:   "dry",
			Usage:  "Enable dry run mode.",
			EnvVar: "DRY",
		},
	},
}

func runServer(c *cli.Context) {
	port := c.String("port")

	b, err := newBuilder(c)
	if err != nil {
		log.Fatal(err)
	}

	s := newServer(c, b)
	log.Println("Listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
