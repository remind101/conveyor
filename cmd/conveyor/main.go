package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor"
)

func main() {
	app := cli.NewApp()
	app.Name = "conveyor"
	app.Usage = "Build docker images from GitHub repositories"
	app.Commands = []cli.Command{
		cmdServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newBuilder(c *cli.Context) (conveyor.Builder, error) {
	b, err := conveyor.NewDockerBuilderFromEnv()
	if err != nil {
		return nil, err
	}
	b.DryRun = c.Bool("dry")

	g := conveyor.NewGitHubClient(c.String("github.token"))
	return conveyor.UpdateGitHubCommitStatus(b, g), nil
}

func newServer(c *cli.Context, b conveyor.Builder) http.Handler {
	b = conveyor.BuildAsync(b)
	return conveyor.NewServerWithSecret(b, c.String("github.secret"))
}
