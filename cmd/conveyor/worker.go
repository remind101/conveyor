package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/worker"
)

// flags for the worker.
var workerFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "github.token",
		Value:  "",
		Usage:  "GitHub API token to use when updating commit statuses and setting up webhooks on repositories.",
		EnvVar: "GITHUB_TOKEN",
	},
	cli.StringFlag{
		Name:   "github.domain",
		Value:  "",
		Usage:  "Github Open Source or Enterprise",
		EnvVar: "GITHUB_DOMAIN",
	}
	cli.BoolFlag{
		Name:   "dry",
		Usage:  "Enable dry run mode.",
		EnvVar: "DRY",
	},
	cli.StringFlag{
		Name:   "builder.image",
		Value:  docker.DefaultBuilderImage,
		Usage:  "A docker image to use to perform the build.",
		EnvVar: "BUILDER_IMAGE",
	},
	cli.StringFlag{
		Name:   "reporter",
		Value:  "",
		Usage:  "The reporter to use to report errors. Available options are `hb://api.honeybadger.io?key=<key>&environment=<environment>",
		EnvVar: "REPORTER",
	},
	cli.IntFlag{
		Name:   "workers",
		Value:  runtime.NumCPU(),
		Usage:  "Number of workers in goroutines to start.",
		EnvVar: "WORKERS",
	},
	cli.StringFlag{
		Name:   "stats",
		Value:  "",
		Usage:  "If provided, defines where build metrics are sent. Available options are dogstatsd://<host>",
		EnvVar: "STATS",
	},
}

var cmdWorker = cli.Command{
	Name:   "worker",
	Usage:  "Run a set of workers.",
	Action: workerAction,
	Flags:  append(sharedFlags, workerFlags...),
}

func workerAction(c *cli.Context) {
	cy := newConveyor(c)

	if err := runWorker(cy, c); err != nil {
		must(err)
	}
}

func runWorker(cy *conveyor.Conveyor, c *cli.Context) error {
	numWorkers := c.Int("workers")

	info("Starting %d workers\n", numWorkers)

	ch := make(chan conveyor.BuildContext)
	cy.BuildQueue.Subscribe(ch)

	workers := worker.NewPool(cy, numWorkers, worker.Options{
		Builder:       newBuilder(c),
		BuildRequests: ch,
	})

	workers.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	info("Signal %d received. Shutting down workers.\n", sig)
	return workers.Shutdown()
}
