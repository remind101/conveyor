package main

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/codegangsta/cli"
	"github.com/ejholmes/hookshot"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
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

// newConveyor builds a new Conveyor instance backed by in memory queue and
// workers.
func newConveyor(c *cli.Context) (*conveyor.Conveyor, error) {
	b, err := newBuilder(c)
	if err != nil {
		return nil, err
	}

	f, err := logFactory(c.String("logger"))
	if err != nil {
		return nil, err
	}

	return conveyor.New(conveyor.Options{
		Builder:    b,
		LogFactory: f,
	}), nil
}

func newBuilder(c *cli.Context) (*conveyor.Builder, error) {
	db, err := docker.NewBuilderFromEnv()
	if err != nil {
		return nil, err
	}
	db.DryRun = c.Bool("dry")
	db.Image = c.String("builder.image")

	g := builder.NewGitHubClient(c.String("github.token"))
	b := conveyor.NewBuilder(builder.UpdateGitHubCommitStatus(db, g))

	r, err := newReporter(c.String("reporter"))
	if err != nil {
		return nil, err
	}
	b.Reporter = r

	return b, nil
}

func newServer(c *cli.Context, b *conveyor.Conveyor) (http.Handler, error) {
	s := conveyor.NewServer(b)
	return hookshot.Authorize(s, c.String("github.secret")), nil
}

func logFactory(uri string) (f builder.LogFactory, err error) {
	var u *url.URL
	u, err = url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "s3":
		f, err = builder.S3Logger(u.Host)
	}

	// f = conveyor.MultiLogger(conveyor.StdoutLogger, f)
	return
}

func newReporter(uri string) (r reporter.Reporter, err error) {
	var u *url.URL
	u, err = url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "hb":
		q := u.Query()
		r = hb2.NewReporter(hb2.Config{
			ApiKey:      q.Get("key"),
			Environment: q.Get("environment"),
		})
	}

	return
}
