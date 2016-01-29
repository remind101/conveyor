package conveyor

import (
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGitHub_InstallHook(t *testing.T) {
	r := new(mockRepositoriesService)
	g := &GitHub{
		Repositories: r,
	}

	hook := NewHook("http://localhost", "secret")

	r.On("ListHooks", "remind101", "acme-inc").Return([]github.Hook{}, nil)
	r.On("CreateHook", "remind101", "acme-inc", hook).Return(nil)

	err := g.InstallHook("remind101", "acme-inc", hook)
	assert.NoError(t, err)

	r.AssertExpectations(t)
}

func TestGitHub_InstallHook_Edit(t *testing.T) {
	r := new(mockRepositoriesService)
	g := &GitHub{
		Repositories: r,
	}

	existingHook := NewHook("http://localhost", "secret")
	existingHook.ID = github.Int(1)

	hook := NewHook("http://localhost", "secret")

	r.On("ListHooks", "remind101", "acme-inc").Return([]github.Hook{*existingHook}, nil)
	r.On("EditHook", "remind101", "acme-inc", 1, hook).Return(nil)

	err := g.InstallHook("remind101", "acme-inc", hook)
	assert.NoError(t, err)

	r.AssertExpectations(t)
}

type mockRepositoriesService struct {
	mock.Mock
}

func (m *mockRepositoriesService) CreateHook(owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := m.Called(owner, repo, hook)
	return nil, nil, args.Error(0)
}

func (m *mockRepositoriesService) ListHooks(owner, repo string, opt *github.ListOptions) ([]github.Hook, *github.Response, error) {
	args := m.Called(owner, repo)
	return args.Get(0).([]github.Hook), nil, args.Error(1)
}

func (m *mockRepositoriesService) EditHook(owner, repo string, id int, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := m.Called(owner, repo, id, hook)
	return nil, nil, args.Error(0)
}
