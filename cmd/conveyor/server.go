package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codegangsta/cli"
	dockerbuilder "github.com/remind101/conveyor/builder/docker"
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
			Value:  dockerbuilder.DefaultBuilderImage,
			Usage:  "A docker image to use to perform the build.",
			EnvVar: "BUILDER_IMAGE",
		},
		cli.StringFlag{
			Name:   "logger",
			Value:  "stdout://",
			Usage:  "The logger to use. Available options are `stdout://`, or `s3://bucket`.",
			EnvVar: "LOGGER",
		},
		cli.StringFlag{
			Name:   "reporter",
			Value:  "",
			Usage:  "The reporter to use to report errors. Available options are `hb://api.honeybadger.io?key=<key>&environment=<environment>",
			EnvVar: "REPORTER",
		},
	},
}

func runServer(c *cli.Context) {
	port := c.String("port")

	b, err := newConveyor(c)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-quit

		log.Printf("Signal %d received. Shutting down.\n", sig)
		if err := b.Shutdown(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	s, err := newServer(c, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
