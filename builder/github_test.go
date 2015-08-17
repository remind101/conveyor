package builder

import (
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
)

type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) CreateStatus(owner, repo, ref string, status *github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	args := m.Called(owner, repo, ref, status)
	return nil, nil, args.Error(0)
}
