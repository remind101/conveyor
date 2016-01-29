// Package slack provides an slash Handler for adding the Conveyor push webhook
// on the GitHub repo.
package slack

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"

	"golang.org/x/net/context"

	"github.com/ejholmes/slash"
	"github.com/remind101/conveyor"
)

// client mocks out the interface from conveyor.Conveyor that we use.
type client interface {
	Build(context.Context, conveyor.BuildRequest) (*conveyor.Build, error)
	EnableRepo(context.Context, string) error
}

// Slack represents the slack slash commands that Conveyor exposes.
type Slack struct {
	// URLTemplate is a template used to return the URL to view the logs for
	// a build.
	URLTemplate *template.Template

	client

	mux slash.Handler
}

// New returns a new Slack instance.
func New(c *conveyor.Conveyor) *Slack {
	return newSlack(c)
}

func newSlack(c client) *Slack {
	r := slash.NewMux()
	s := &Slack{client: c, mux: r}

	r.Match(slash.MatchSubcommand(`help`), Help)
	r.MatchText(
		regexp.MustCompile(`enable (?P<owner>\S+?)/(?P<repo>\S+)`),
		slash.HandlerFunc(s.Enable),
	)
	r.MatchText(
		regexp.MustCompile(`build (?P<owner>\S+?)/(?P<repo>\S+)@(?P<branch>\S+)`),
		slash.HandlerFunc(s.Build),
	)

	return s
}

func (s *Slack) ServeCommand(ctx context.Context, r slash.Responder, command slash.Command) error {
	return s.mux.ServeCommand(ctx, r, command)
}

func (s *Slack) Build(ctx context.Context, r slash.Responder, command slash.Command) error {
	params := slash.Params(ctx)

	owner, repo, branch := params["owner"], params["repo"], params["branch"]

	r.Respond(slash.Reply("One moment..."))

	if err := s.build(ctx, r, owner, repo, branch); err != nil {
		return r.Respond(slash.Reply(fmt.Sprintf("error: %s", err)))
	}

	return nil
}

func (s *Slack) build(ctx context.Context, r slash.Responder, owner, repo, branch string) error {
	fullRepo := fmt.Sprintf("%s/%s", owner, repo)

	build, err := s.client.Build(ctx, conveyor.BuildRequest{
		Repository: fullRepo,
		Branch:     branch,
	})
	if err != nil {
		return err
	}

	url, err := s.url(build)
	if err != nil {
		return err
	}

	return r.Respond(slash.Reply(fmt.Sprintf("Building %s@%s: %s", fullRepo, branch, url)))
}

func (s *Slack) url(build *conveyor.Build) (string, error) {
	t := s.URLTemplate
	buf := new(bytes.Buffer)
	err := t.Execute(buf, build)
	return buf.String(), err
}

func (s *Slack) Enable(ctx context.Context, r slash.Responder, command slash.Command) error {
	params := slash.Params(ctx)
	owner, repo := params["owner"], params["repo"]

	if err := s.client.EnableRepo(ctx, fmt.Sprintf("%s/%s", owner, repo)); err != nil {
		return r.Respond(slash.Reply(fmt.Sprintf("error: %v", err)))
	}

	return r.Respond(slash.Reply(fmt.Sprintf("Installed webhook on %s/%s", owner, repo)))
}

// replyHandler returns a slash.Handler that just replies to the user with the
// text.
func replyHandler(text string) slash.Handler {
	return slash.HandlerFunc(func(ctx context.Context, r slash.Responder, c slash.Command) error {
		return r.Respond(slash.Reply(text))
	})
}
