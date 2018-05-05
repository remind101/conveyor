package builder

import (
	"context"

	"github.com/google/go-github/github"
)

// GitHubClient represents a client that can create github commit statuses.
type GitHubClient interface {
	CreateStatus(context context.Context, owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}

// NewGitHubClient returns a new GitHubClient instance. If token is an empty
// string, then a fake client will be returned.
func NewGitHubClient(c *github.Client) GitHubClient {
	return c.Repositories
}

// nullGitHubClient is an implementation of the GitHubClient interface that does
// nothing.
type nullGitHubClient struct{}

func (c *nullGitHubClient) CreateStatus(ctx context.Context, owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	return nil, nil, nil
}
