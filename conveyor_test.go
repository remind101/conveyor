package conveyor

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

func TestConveyor_Build(t *testing.T) {
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", nil
	}
	l := &mockLogger{}
	c := New(BuilderFunc(b))

	if _, err := c.Build(context.Background(), BuildOptions{
		OutputStream: l,
	}); err != nil {
		t.Fatal(err)
	}

	if !l.closed {
		t.Fatal("Expected logger to be closed")
	}
}

func TestConveyor_Build_CloseError(t *testing.T) {
	closeErr := errors.New("i/o timeout")
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", nil
	}
	l := &mockLogger{closeErr: closeErr}
	c := New(BuilderFunc(b))

	if _, err := c.Build(context.Background(), BuildOptions{
		OutputStream: l,
	}); err != closeErr {
		t.Fatalf("Expected error to be %v", closeErr)
	}

	if !l.closed {
		t.Fatal("Expected logger to be closed")
	}
}

func TestUpdateGitHubCommitStatus(t *testing.T) {
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", nil
	}
	g := &MockGitHubClient{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)
	builder.since = func(t time.Time) time.Duration {
		return time.Second
	}

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("pending"),
		Description: github.String("Image building."),
		TargetURL:   github.String(""),
		Context:     github.String("container/docker"),
	}).Return(nil)
	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("success"),
		Description: github.String("Image built in 1s."),
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

func TestUpdateGitHubCommitStatus_Error(t *testing.T) {
	b := func(ctx context.Context, opts BuildOptions) (string, error) {
		return "", errors.New("i/o timeout")
	}
	g := &MockGitHubClient{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)
	builder.since = func(t time.Time) time.Duration {
		return time.Second
	}

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("pending"),
		Description: github.String("Image building."),
		TargetURL:   github.String(""),
		Context:     github.String("container/docker"),
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

type mockLogger struct {
	closeErr error
	closed   bool
}

func (m *mockLogger) Write(p []byte) (int, error) {
	return len(p), nil
}

func (m *mockLogger) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockLogger) URL() string {
	return "https://google.com"
}
