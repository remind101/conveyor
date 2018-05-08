package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/goji/httpauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/internal/ghinstallation"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/conveyor/logs/cloudwatch"
	"github.com/remind101/conveyor/logs/s3"
	"github.com/remind101/conveyor/server"
	"github.com/remind101/conveyor/worker"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
)

const logsURLTemplate = "%s/logs/{{.ID}}"

func newDB(c *cli.Context) *sqlx.DB {
	db := sqlx.MustConnect("postgres", c.String("db"))
	if err := conveyor.MigrateUp(db); err != nil {
		panic(err)
	}
	return db
}

func newConveyor(c *cli.Context) *conveyor.Conveyor {
	cy := conveyor.New(newDB(c))
	cy.BuildQueue = newBuildQueue(c)
	cy.Logger = newLogger(c)
	cy.GitHub = conveyor.NewGitHub(newGitHubClient(c))
	return cy
}

func newBuildQueue(c *cli.Context) conveyor.BuildQueue {
	u := urlParse(c.String("queue"))

	switch u.Scheme {
	case "memory":
		return conveyor.NewBuildQueue(100)
	case "sqs":
		q := conveyor.NewSQSBuildQueue(defaults.DefaultConfig)
		if u.Host == "" {
			q.QueueURL = os.Getenv("SQS_QUEUE_URL")
		} else {
			url := *u
			url.Scheme = "https"
			q.QueueURL = url.String()
		}
		return q
	default:
		must(fmt.Errorf("Unknown queue: %v", u.Scheme))
		return nil
	}
}

func newServer(cy *conveyor.Conveyor, c *cli.Context) http.Handler {
	var apiAuth func(http.Handler) http.Handler

	if auth := c.String("auth"); auth != "" {
		parts := strings.Split(auth, ":")
		apiAuth = httpauth.SimpleBasicAuth(parts[0], parts[1])
	} else {
		apiAuth = func(h http.Handler) http.Handler { return h }
	}

	r := mux.NewRouter()
	r.NotFoundHandler = server.NewServer(cy, server.Config{
		APIAuth:      apiAuth,
		GitHubSecret: c.String("github.secret"),
	})

	n := negroni.Classic()
	n.UseHandler(r)

	return n
}

func newGitHubClient(c *cli.Context) *github.Client {
	t, err := ghinstallation.New(http.DefaultTransport, c.Int("github.app_id"), c.Int("github.installation_id"), []byte(c.String("github.private_key")))
	if err != nil {
		panic(err)
	}

	return github.NewClient(&http.Client{Transport: t})
}

func newBuilder(c *cli.Context) builder.Builder {
	db, err := docker.NewBuilderFromEnv()
	if err != nil {
		must(err)
	}
	db.DryRun = c.Bool("dry")
	db.Image = c.String("builder.image")

	g := builder.NewGitHubClient(newGitHubClient(c))

	var backend builder.Builder = builder.UpdateGitHubCommitStatus(db, g, fmt.Sprintf(logsURLTemplate, c.String("url")))

	b := worker.NewBuilder(backend)
	b.Reporter = newReporter(c)

	return b
}

func newReporter(c *cli.Context) reporter.Reporter {
	u := urlParse(c.String("reporter"))

	switch u.Scheme {
	case "hb":
		q := u.Query()
		return hb2.NewReporter(hb2.Config{
			ApiKey:      q.Get("key"),
			Environment: q.Get("environment"),
		})
	default:
		return nil
	}
}

func newLogger(c *cli.Context) logs.Logger {
	u := urlParse(c.String("logger"))

	switch u.Scheme {
	case "s3":
		return s3.NewLogger(u.Host)
	case "cloudwatch":
		return cloudwatch.NewLogger(u.Host)
	case "stdout":
		return logs.Stdout
	default:
		must(fmt.Errorf("Unknown logger: %v", u.Scheme))
		return nil
	}
}

func urlParse(uri string) *url.URL {
	u, err := url.Parse(uri)
	if err != nil {
		must(err)
	}
	return u
}
