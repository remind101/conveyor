package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"golang.org/x/oauth2"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/ejholmes/slash"
	"github.com/goji/httpauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/datadog"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/conveyor/logs/cloudwatch"
	"github.com/remind101/conveyor/logs/s3"
	"github.com/remind101/conveyor/server"
	"github.com/remind101/conveyor/slack"
	"github.com/remind101/conveyor/worker"
	"github.com/remind101/pkg/reporter/config"
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
	cy.Hook = conveyor.NewHook(c.String("url"), c.String("github.secret"))
	return cy
}

func newBuildQueue(c *cli.Context) conveyor.BuildQueue {
	u := urlParse(c.String("queue"))

	switch u.Scheme {
	case "memory":
		return conveyor.NewBuildQueue(100)
	case "sqs":
		q := conveyor.NewSQSBuildQueue(session.New())
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

	// Slack webhooks
	if c.String("slack.token") != "" {
		r.Handle("/slack", newSlackServer(cy, c))
	}

	n := negroni.Classic()
	n.UseHandler(r)

	return n
}

func newGitHubClient(c *cli.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.String("github.token")},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return github.NewClient(tc)
}

// newSlackServer returns an http handler for handling Slack slash commands at <url>/slack.
func newSlackServer(cy *conveyor.Conveyor, c *cli.Context) http.Handler {
	s := slack.New(cy)
	s.URLTemplate = template.Must(template.New("url").Parse(fmt.Sprintf(logsURLTemplate, c.String("url"))))
	return slash.NewServer(slash.ValidateToken(s, c.String("slack.token")))
}

func newBuilder(c *cli.Context) builder.Builder {
	db, err := docker.NewBuilderFromEnv()
	if err != nil {
		must(err)
	}
	db.DryRun = c.Bool("dry")
	db.Image = c.String("builder.image")

	g := builder.NewGitHubClient(c.String("github.token"))

	var backend builder.Builder = builder.UpdateGitHubCommitStatus(db, g, fmt.Sprintf(logsURLTemplate, c.String("url")))

	if uri := c.String("stats"); uri != "" {
		u := urlParse(uri)

		switch u.Scheme {
		case "dogstatsd":
			c, err := statsd.New(u.Host)
			must(err)

			backend = datadog.WithStats(
				backend,
				c,
			)
		default:
			must(fmt.Errorf("Unknown stats backend: %v", u.Scheme))
		}
	}

	b := worker.NewBuilder(backend)
	b.Reporter, err = config.NewReporterFromUrls(c.StringSlice("reporter"))
	must(err)

	return b
}

func newLogger(c *cli.Context) logs.Logger {
	u := urlParse(c.String("logger"))

	switch u.Scheme {
	case "s3":
		return s3.NewLogger(session.New(), u.Host)
	case "cloudwatch":
		return cloudwatch.NewLogger(session.New(), u.Host)
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
