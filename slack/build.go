package slack

import (
	"fmt"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ejholmes/slash"
	"github.com/google/go-github/github"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// newID returns a new unique identifier.
var newID = uuid.New

type branchResolver interface {
	resolveBranch(owner, repo, branch string) (sha string, err error)
}

// Build is a slash.Handler that will trigger a conveyor build.
type Build struct {
	// BuildQueue to use.
	Queue conveyor.BuildQueue

	branchResolver
}

func NewBuild(client *github.Client, q conveyor.BuildQueue) *Build {
	return &Build{
		Queue:          q,
		branchResolver: &githubBranchResolver{client.Git},
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

	id := newID()
	opts := builder.BuildOptions{
		ID:         id,
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		Branch:     branch,
		Sha:        sha,
	}
	if err := b.Queue.Push(ctx, opts); err != nil {
		return r.Respond(slash.Reply(err.Error()))
	}

	return r.Respond(slash.Reply("Build enqueued"))
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
