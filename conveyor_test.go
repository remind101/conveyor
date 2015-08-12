package conveyor

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

func TestConveyor_Build(t *testing.T) {
	b := func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		return "", nil
	}
	w := &mockLogger{}
	c := New(BuilderFunc(b))

	if _, err := c.Build(context.Background(), w, BuildOptions{}); err != nil {
		t.Fatal(err)
	}

	if !w.closed {
		t.Fatal("Expected logger to be closed")
	}
}

func TestConveyor_Build_CloseError(t *testing.T) {
	closeErr := errors.New("i/o timeout")
	b := func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		return "", nil
	}
	w := &mockLogger{closeErr: closeErr}
	c := New(BuilderFunc(b))

	if _, err := c.Build(context.Background(), w, BuildOptions{}); err != closeErr {
		t.Fatalf("Expected error to be %v", closeErr)
	}

	if !w.closed {
		t.Fatal("Expected logger to be closed")
	}
}

func TestUpdateGitHubCommitStatus(t *testing.T) {
	b := func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		return "", nil
	}
	g := &MockGitHubClient{}
	w := &mockLogger{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)
	builder.since = func(t time.Time) time.Duration {
		return time.Second
	}

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("pending"),
		Description: github.String("Image building."),
		Context:     github.String("container/docker"),
	}).Return(nil)
	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("success"),
		Description: github.String("Image built in 1s."),
		TargetURL:   github.String("https://google.com"),
		Context:     github.String("container/docker"),
	}).Return(nil)

	builder.Build(context.Background(), w, BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	})

	g.AssertExpectations(t)
}

func TestUpdateGitHubCommitStatus_Error(t *testing.T) {
	b := func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		return "", errors.New("i/o timeout")
	}
	g := &MockGitHubClient{}
	w := &mockLogger{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)
	builder.since = func(t time.Time) time.Duration {
		return time.Second
	}

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("pending"),
		Description: github.String("Image building."),
		Context:     github.String("container/docker"),
	}).Return(nil)
	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("failure"),
		Description: github.String("i/o timeout"),
		TargetURL:   github.String("https://google.com"),
		Context:     github.String("container/docker"),
	}).Return(nil)

	builder.Build(context.Background(), w, BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	})

	g.AssertExpectations(t)
}

func TestUpdateGitHubCommitStatus_CommitNotFound(t *testing.T) {
	var called bool
	b := func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		called = true
		return "", nil
	}
	g := &MockGitHubClient{}
	w := &mockLogger{}
	builder := UpdateGitHubCommitStatus(BuilderFunc(b), g)

	g.On("CreateStatus", "remind101", "acme-inc", "abcd", &github.RepoStatus{
		State:       github.String("pending"),
		Description: github.String("Image building."),
		Context:     github.String("container/docker"),
	}).Return(&github.ErrorResponse{
		Response: &http.Response{
			StatusCode: 404,
		},
	})

	builder.Build(context.Background(), w, BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	})

	g.AssertExpectations(t)

	if called {
		t.Fatal("Expected builder to not be called")
	}
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
