package events

import (
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func init() {
	since = func(time.Time) time.Duration {
		return time.Second
	}
}

func TestBuilder_Build(t *testing.T) {
	e := new(mockBuildEvents)
	b := Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", nil
		}),
		events: e,
	}

	options := builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}

	e.On("BuildEvent", &BuildStartedEvent{
		BuildOptions: options,
	}).Return(nil)

	e.On("BuildEvent", &BuildCompletedEvent{
		BuildOptions: options,
		Duration:     time.Second,
		Image:        "",
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, options)
	assert.NoError(t, err)
}

func TestBuilder_Build_Err(t *testing.T) {
	errBoom := errors.New("boom")

	e := new(mockBuildEvents)
	b := Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", errBoom
		}),
		events: e,
	}

	options := builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}

	e.On("BuildEvent", &BuildStartedEvent{
		BuildOptions: options,
	}).Return(nil)

	e.On("BuildEvent", &BuildCompletedEvent{
		BuildOptions: options,
		Duration:     time.Second,
		Image:        "",
		Err:          errBoom,
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, options)
	assert.Equal(t, err, errBoom)
}

func TestBuilder_Build_LogsURL(t *testing.T) {
	e := new(mockBuildEvents)
	b := Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", nil
		}),
		events: e,
	}

	options := builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}

	e.On("BuildEvent", &BuildStartedEvent{
		BuildOptions: options,
	}).Return(nil)

	e.On("BuildEvent", &BuildCompletedEvent{
		BuildOptions: options,
		Duration:     time.Second,
		Image:        "",
		Logs:         "http://www.google.com",
	}).Return(nil)

	_, err := b.Build(context.Background(), new(mockLogger), options)
	assert.NoError(t, err)
}

type mockBuildEvents struct {
	mock.Mock
}

func (e *mockBuildEvents) BuildEvent(event interface{}) error {
	args := e.Called(event)
	return args.Error(0)
}

type mockLogger struct {
	io.Writer
}

func (l *mockLogger) URL() string {
	return "http://www.google.com"
}
