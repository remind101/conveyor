package main

import (
	"net/http"
	"net/url"

	"github.com/codegangsta/cli"
	"github.com/ejholmes/hookshot"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
)

func newBuildQueue(c *cli.Context) conveyor.BuildQueue {
	return conveyor.NewBuildQueue(100)
}

func newServer(q conveyor.BuildQueue, c *cli.Context) http.Handler {
	return hookshot.Authorize(
		conveyor.NewServer(q),
		c.String("github.secret"),
	)
}

func newBuilder(c *cli.Context) builder.Builder {
	db, err := docker.NewBuilderFromEnv()
	if err != nil {
		must(err)
	}
	db.DryRun = c.Bool("dry")
	db.Image = c.String("builder.image")

	g := builder.NewGitHubClient(c.String("github.token"))
	b := conveyor.NewBuilder(builder.UpdateGitHubCommitStatus(db, g))
	b.Reporter = newReporter(c)
	return b
}

func newReporter(c *cli.Context) reporter.Reporter {
	u, err := url.Parse(c.String("reporter"))
	if err != nil {
		must(err)
	}

	switch u.Scheme {
	case "hb":
		q := u.Query()
		return hb2.NewReporter(hb2.Config{
			ApiKey:      q.Get("key"),
			Environment: q.Get("environment"),
		})
	default:
		must(err)
		return nil
	}
}

func newLogFactory(c *cli.Context) builder.LogFactory {
	u, err := url.Parse(c.String("logger"))
	if err != nil {
		must(err)
	}

	switch u.Scheme {
	case "s3":
		f, err := builder.S3Logger(u.Host)
		if err != nil {
			must(err)
		}
		return f
	default:
		must(err)
		return nil
	}
}
