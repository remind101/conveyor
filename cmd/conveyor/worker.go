package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/worker"
)

const hbExampleURL = "hb://api.honeybadger.io?key=<key>&environment=<environment>"
const rollbarExampleURL = "rollbar://api.rollbar.com?key=<key>&environment=<environment>"

// flags for the worker.
var workerFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "github.token",
		Value:  "",
		Usage:  "GitHub API token to use when updating commit statuses and setting up webhooks on repositories.",
		EnvVar: "GITHUB_TOKEN",
	},
	cli.BoolFlag{
		Name:   "dry",
		Usage:  "Enable dry run mode.",
		EnvVar: "DRY",
	},
	cli.StringFlag{
		Name:   "builder",
		Value:  "codebuild",
		Usage:  "The builder backend to use. Available options are `codebuild` or `docker`",
		EnvVar: "BUILDER",
	},
	cli.StringFlag{
		Name:   "builder.image",
		Value:  docker.DefaultBuilderImage,
		Usage:  "A docker image to use to perform the build. Only used with the Docker backend.",
		EnvVar: "BUILDER_IMAGE",
	},
	cli.StringFlag{
		Name:   "codebuild.role",
		Value:  "",
		Usage:  "An IAM role to provide as the service role to CodeBuild projects.",
		EnvVar: "CODEBUILD_SERVICE_ROLE",
	},
	cli.StringFlag{
		Name:   "codebuild.project.prefix",
		Value:  "conveyor-",
		Usage:  "A prefix that Conveyor will use when creating CodeBuild projects for repos.",
		EnvVar: "CODEBUILD_PROJECT_PREFIX",
	},
	cli.StringFlag{
		Name:   "codebuild.dockercfg",
		Value:  "conveyor.dockercfg",
		Usage:  "An SSM parameter that CodeBuild will use to authenticate the docker cli. Should be a valid .dockercfg file.",
		EnvVar: "CODEBUILD_DOCKERCFG",
	},
	cli.StringSliceFlag{
		Name:  "reporter",
		Value: &cli.StringSlice{},
		Usage: fmt.Sprintf("The reporter to use to report errors. Available options are `%s` or `%s`",
			hbExampleURL, rollbarExampleURL),
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
		Builder:       newWorkerBuilder(c),
		BuildRequests: ch,
	})

	workers.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	info("Signal %d received. Shutting down workers.\n", sig)
	return workers.Shutdown()
}
