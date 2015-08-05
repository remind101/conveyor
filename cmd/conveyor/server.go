package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
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
		cli.StringFlag{
			Name:   "builder.image",
			Value:  conveyor.DefaultBuilderImage,
			Usage:  "A docker image to use to perform the build.",
			EnvVar: "BUILDER_IMAGE",
		},
		cli.StringFlag{
			Name:   "logger",
			Value:  "stdout://",
			Usage:  "The logger to use. Available options are `stdout://`, or `s3://bucket`.",
			EnvVar: "LOGGER",
		},
	},
}

func runServer(c *cli.Context) {
	port := c.String("port")

	b, err := newBuilder(c)
	if err != nil {
		log.Fatal(err)
	}

	s, err := newServer(c, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
