package conveyor

import (
	"errors"
	"testing"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

func TestUpdateGitHubCommitStatus(t *testing.T) {
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", nil
	}
	g := &MockGitHubClient{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:     github.String("pending"),
		TargetURL: github.String(""),
		Context:   github.String("container/docker"),
	}).Return(nil)
	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:     github.String("success"),
		TargetURL: github.String(""),
		Context:   github.String("container/docker"),
	}).Return(nil)

	builder.Build(context.Background(), BuildOptions{
		Repository:   "remind101/acme-inc",
		Branch:       "master",
		Sha:          "abcd",
		OutputStream: &logger{},
	})

	g.AssertExpectations(t)
}

func TestUpdateGitHubCommitStatus_Error(t *testing.T) {
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", errors.New("i/o timeout")
	}
	g := &MockGitHubClient{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:     github.String("pending"),
		TargetURL: github.String(""),
		Context:   github.String("container/docker"),
	}).Return(nil)
	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("failure"),
		Description: github.String("i/o timeout"),
		TargetURL:   github.String(""),
		Context:     github.String("container/docker"),
	}).Return(nil)

	builder.Build(context.Background(), BuildOptions{
		Repository:   "remind101/acme-inc",
		Branch:       "master",
		Sha:          "abcd",
		OutputStream: &logger{},
	})

	g.AssertExpectations(t)
}
