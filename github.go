package conveyor

import (
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// GitHubClient represents a client that can create github commit statuses.
type GitHubClient interface {
	CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
}

// NewGitHubClient returns a new GitHubClient instance. If token is an empty
// string, then a fake client will be returned.
func NewGitHubClient(token string) GitHubClient {
	if token == "" {
		return &nullGitHubClient{}
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc).Repositories
}

// nullGitHubClient is an implementation of the GitHubClient interface that does
// nothing.
type nullGitHubClient struct{}

func (c *nullGitHubClient) CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	return nil, nil, nil
}
