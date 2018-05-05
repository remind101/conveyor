package conveyor

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
)

// GitHubAPI represents an interface for performing Git operations.
type GitHubAPI interface {
	ResolveBranch(ctx context.Context, owner, repo, branch string) (sha string, err error)
}

func NewGitHub(c *github.Client) *GitHub {
	return &GitHub{
		Git: c.Git,
	}
}

// GitHub is an implementation of the Git interface
// backed by the GitHub API.
type GitHub struct {
	Git *github.GitService
}

func (g *GitHub) ResolveBranch(ctx context.Context, owner, repo, branch string) (string, error) {
	ref, _, err := g.Git.GetRef(ctx, owner, repo, fmt.Sprintf("refs/heads/%s", branch))
	if err != nil {
		return "", err
	}
	return *ref.Object.SHA, nil
}

func splitRepo(fullRepo string) (owner, repo string) {
	parts := strings.Split(fullRepo, "/")
	owner, repo = parts[0], parts[1]
	return
}
