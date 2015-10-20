package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder/docker"
)

// flags for the worker.
var workerFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "github.token",
		Value:  "",
		Usage:  "GitHub API token to use when updating commit statuses on repositories.",
		EnvVar: "GITHUB_TOKEN",
	},
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
	cli.IntFlag{
		Name:   "workers",
		Value:  runtime.NumCPU(),
		Usage:  "Number of workers in goroutines to start.",
		EnvVar: "WORKERS",
	},
}

var cmdWorker = cli.Command{
	Name:   "worker",
	Usage:  "Run a set of workers.",
	Action: workerAction,
	Flags:  append(sharedFlags, workerFlags...),
}

func workerAction(c *cli.Context) {
	q := newBuildQueue(c)

	if err := runWorker(q, c); err != nil {
		must(err)
	}
}

func runWorker(q conveyor.BuildQueue, c *cli.Context) error {
	numWorkers := c.Int("workers")

	info("Starting %d workers\n", numWorkers)

	ch := make(chan conveyor.BuildRequest)
	go func() {
		for {
			if err := q.Subscribe(ch); err != nil {
				info("queue error: %v", err)
			}
		}
	}()

	workers := conveyor.NewWorkerPool(numWorkers, conveyor.WorkerOptions{
		Builder:       newBuilder(c),
		LogFactory:    newLogFactory(c),
		BuildRequests: ch,
	})

	workers.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	info("Signal %d received. Shutting down workers.\n", sig)
	return workers.Shutdown()
}
