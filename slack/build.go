package slack

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/ejholmes/slash"
	"github.com/google/go-github/github"
	"github.com/remind101/conveyor"
	"golang.org/x/net/context"
)

type branchResolver interface {
	resolveBranch(owner, repo, branch string) (sha string, err error)
}

// Build is a slash.Handler that will trigger a conveyor build.
type Build struct {
	client

	branchResolver
	urlTmpl *template.Template
}

func NewBuild(client *github.Client, c *conveyor.Conveyor, urlTmpl string) *Build {
	return &Build{
		client:         c,
		branchResolver: &githubBranchResolver{client.Git},
		urlTmpl:        template.Must(template.New("url").Parse(urlTmpl)),
	}
}

func (b *Build) ServeCommand(ctx context.Context, r slash.Responder, c slash.Command) (slash.Response, error) {
	params := slash.Params(ctx)

	owner, repo, branch := params["owner"], params["repo"], params["branch"]
	go b.build(ctx, r, owner, repo, branch)

	return slash.Reply("One moment..."), nil
}

func (b *Build) build(ctx context.Context, r slash.Responder, owner, repo, branch string) error {
	sha, err := b.resolveBranch(owner, repo, branch)
	if err != nil {
		return r.Respond(slash.Reply(err.Error()))
	}

	fullRepo := fmt.Sprintf("%s/%s", owner, repo)
	req := conveyor.BuildRequest{
		Repository: fullRepo,
		Branch:     branch,
		Sha:        sha,
	}
	build, err := b.client.Build(ctx, req)
	if err != nil {
		return r.Respond(slash.Reply(err.Error()))
	}

	url, err := b.url(build)
	if err != nil {
		return r.Respond(slash.Reply(err.Error()))
	}

	return r.Respond(slash.Reply(fmt.Sprintf("Building %s@%s: %s", fullRepo, branch, url)))
}

func (b *Build) url(build *conveyor.Build) (string, error) {
	buf := new(bytes.Buffer)
	err := b.urlTmpl.Execute(buf, build)
	return buf.String(), err
}

type githubBranchResolver struct {
	git *github.GitService
}

func (r *githubBranchResolver) resolveBranch(owner, repo, branch string) (string, error) {
	ref, _, err := r.git.GetRef(owner, repo, fmt.Sprintf("refs/heads/%s", branch))
	if err != nil {
		return "", err
	}
	return *ref.Object.SHA, nil
}
