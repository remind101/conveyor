package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"golang.org/x/oauth2"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/ejholmes/slash"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/datadog"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/conveyor/logs/cloudwatch"
	"github.com/remind101/conveyor/logs/s3"
	"github.com/remind101/conveyor/server"
	"github.com/remind101/conveyor/slack"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
)

const logsURLTemplate = "%s/logs/{{.ID}}"

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

func newServer(q conveyor.BuildQueue, c *cli.Context) http.Handler {
	r := mux.NewRouter()
	r.NotFoundHandler = server.NewServer(server.Config{
		GitHubSecret: c.String("github.secret"),
		Queue:        q,
		Logger:       newLogger(c),
	})

	// Slack webhooks
	if c.String("slack.token") != "" {
		r.Handle("/slack", newSlackServer(q, c))
	}

	n := negroni.Classic()
	n.UseHandler(r)

	return n
}

// newSlackServer returns an http handler for handling Slack slash commands at <url>/slack.
func newSlackServer(q conveyor.BuildQueue, c *cli.Context) http.Handler {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.String("github.token")},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	r := slash.NewMux()
	r.Match(slash.MatchSubcommand(`help`), slack.Help)
	r.MatchText(
		regexp.MustCompile(`enable (?P<owner>\S+?)/(?P<repo>\S+)`),
		slack.NewEnable(
			client,
			slack.NewHook(c.String("url"), c.String("github.secret")),
		),
	)
	r.MatchText(
		slack.BuildRegexp,
		slack.NewBuild(
			client,
			q,
			fmt.Sprintf(logsURLTemplate, c.String("url")),
		),
	)

	return slash.NewServer(slash.ValidateToken(r, c.String("slack.token")))
}

func newBuilder(c *cli.Context) builder.Builder {
	db, err := docker.NewBuilderFromEnv()
	if err != nil {
		must(err)
	}
	db.DryRun = c.Bool("dry")
	db.Image = c.String("builder.image")

	g := builder.NewGitHubClient(c.String("github.token"))

	var backend builder.Builder
	backend = builder.UpdateGitHubCommitStatus(db, g, fmt.Sprintf(logsURLTemplate, c.String("url")))

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

	b := conveyor.NewBuilder(backend)
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
