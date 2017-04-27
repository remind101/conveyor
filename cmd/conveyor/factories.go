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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/ejholmes/slash"
	"github.com/goji/httpauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/codebuild"
	"github.com/remind101/conveyor/builder/datadog"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/conveyor/logs/cloudwatch"
	"github.com/remind101/conveyor/logs/s3"
	"github.com/remind101/conveyor/server"
	"github.com/remind101/conveyor/slack"
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

func selectBuilder(c *cli.Context) builder.Builder {
	selectedBuilder := c.String("builder")

	switch selectedBuilder {
	case "docker":
		db, err := docker.NewBuilderFromEnv()
		if err != nil {
			must(err)
		}
		db.DryRun = c.Bool("dry")
		db.Image = c.String("builder.image")

		return db

	case "codebuild":
		cb := codebuild.NewBuilder(session.Must(session.NewSession()))

		// Codebuild configs
		cb.ServiceRole = c.String("builder.codebuild.serviceRole")
		cb.ComputeType = c.String("builder.codebuild.computeType")
		cb.Image = c.String("builder.image")

		// // Dockerhub configuration
		// cb.DockerUsername = c.String("docker.username")
		// cb.DockerPassword = c.String("docker.password")

		// Add secrets to SSM store
		s := ssm.New(session.Must(session.NewSession()))

		password := &ssm.PutParameterInput{
			Name:      aws.String("conveyor.dockerusername"),   // Required
			Type:      aws.String("SecureString"),              // Required
			Value:     aws.String(c.String("docker.username")), // Required
			KeyId:     aws.String(c.String("key.arn")),
			Overwrite: aws.Bool(true),
		}

		username := &ssm.PutParameterInput{
			Name:      aws.String("conveyor.dockerpassword"),   // Required
			Type:      aws.String("SecureString"),              // Required
			Value:     aws.String(c.String("docker.password")), // Required
			KeyId:     aws.String(c.String("key.arn")),
			Overwrite: aws.Bool(true),
		}

		_, err := s.PutParameter(password)

		if err != nil {
			must(err)
		}

		_, err = s.PutParameter(username)

		if err != nil {
			must(err)
		}

		return cb

	default:
		must(fmt.Errorf("Unknown builder: %v", selectedBuilder))
		return nil
	}
}

func newBuilder(c *cli.Context) builder.Builder {

	sb := selectBuilder(c)

	g := builder.NewGitHubClient(c.String("github.token"))

	var backend builder.Builder = builder.UpdateGitHubCommitStatus(sb, g, fmt.Sprintf(logsURLTemplate, c.String("url")))

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

	sess, err := session.NewSession()
	if err != nil {
		must(err)
		return nil
	}

	switch u.Scheme {
	case "s3":
		return s3.NewLogger(sess, u.Host)
	case "cloudwatch":
		return cloudwatch.NewLogger(sess, u.Host)
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
