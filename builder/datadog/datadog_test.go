package datadog

import (
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// mock out since to return 1 second.
	since = func(time.Time) time.Duration {
		return time.Second
	}
}

func TestBuilder_Build(t *testing.T) {
	c := new(mockStatsdClient)
	b := &Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "remind101/acme-inc:1234", nil
		}),
		statsd: c,
	}

	c.On("TimeInMilliseconds",
		"conveyor.build.time",
		float64(1000),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Event", &statsd.Event{
		Title: "Conveyor built remind101/acme-inc:1234",
		Text:  "Built remind101/acme-inc:1234 from remind101/acme-inc@master",
		Tags: []string{
			"repo:remind101/acme-inc",
			"branch:master",
			"sha:1234",
			"image:remind101/acme-inc:1234",
		},
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "1234",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestBuilder_Build_LogURL(t *testing.T) {
	c := new(mockStatsdClient)
	b := &Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "remind101/acme-inc:1234", nil
		}),
		statsd: c,
	}

	c.On("TimeInMilliseconds",
		"conveyor.build.time",
		float64(1000),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Event", &statsd.Event{
		Title: "Conveyor built remind101/acme-inc:1234",
		Text: `Built remind101/acme-inc:1234 from remind101/acme-inc@master

**[View logs](http://www.google.com)**`,
		Tags: []string{
			"repo:remind101/acme-inc",
			"branch:master",
			"sha:1234",
			"image:remind101/acme-inc:1234",
		},
	}).Return(nil)

	_, err := b.Build(context.Background(), new(mockLogger), builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "1234",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestBuilder_Build_Err(t *testing.T) {
	errBoom := errors.New("container returned non-zero exit")

	c := new(mockStatsdClient)
	b := &Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", errBoom
		}),
		statsd: c,
	}

	c.On("TimeInMilliseconds",
		"conveyor.build.time",
		float64(1000),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Count",
		"conveyor.build.error",
		int64(1),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Event", &statsd.Event{
		Title: "Conveyor build failed",
		Text:  "Build of remind101/acme-inc@master failed with: container returned non-zero exit",
		Tags: []string{
			"repo:remind101/acme-inc",
			"branch:master",
			"sha:1234",
		},
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "1234",
		Branch:     "master",
	})
	assert.Equal(t, err, errBoom)

	c.AssertExpectations(t)
}

func TestBuilder_Build_Canceled(t *testing.T) {
	errCanceled := &builder.BuildCanceledError{
		Err:    errors.New("container returned non-zero exit"),
		Reason: context.Canceled,
	}

	c := new(mockStatsdClient)
	b := &Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", errCanceled
		}),
		statsd: c,
	}

	c.On("TimeInMilliseconds",
		"conveyor.build.time",
		float64(1000),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Count",
		"conveyor.build.error",
		int64(1),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)

	c.On("Count",
		"conveyor.build.canceled",
		int64(1),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Event", &statsd.Event{
		Title: "Conveyor build failed",
		Text:  "Build of remind101/acme-inc@master failed with: container returned non-zero exit (context canceled)",
		Tags: []string{
			"repo:remind101/acme-inc",
			"branch:master",
			"sha:1234",
		},
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "1234",
	})
	assert.Equal(t, err, errCanceled)

	c.AssertExpectations(t)
}

func TestBuilder_Build_DeadlineExceeded(t *testing.T) {
	errCanceled := &builder.BuildCanceledError{
		Err:    errors.New("container returned non-zero exit"),
		Reason: context.DeadlineExceeded,
	}

	c := new(mockStatsdClient)
	b := &Builder{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "", errCanceled
		}),
		statsd: c,
	}

	c.On("TimeInMilliseconds",
		"conveyor.build.time",
		float64(1000),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Count",
		"conveyor.build.error",
		int64(1),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Count",
		"conveyor.build.timedout",
		int64(1),
		[]string{"repo:remind101/acme-inc"},
		float64(1),
	).Return(nil)
	c.On("Event", &statsd.Event{
		Title: "Conveyor build failed",
		Text:  "Build of remind101/acme-inc@master failed with: container returned non-zero exit (context deadline exceeded)",
		Tags: []string{
			"repo:remind101/acme-inc",
			"branch:master",
			"sha:1234",
		},
	}).Return(nil)

	_, err := b.Build(context.Background(), ioutil.Discard, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "1234",
	})
	assert.Equal(t, err, errCanceled)

	c.AssertExpectations(t)
}

// mockStatsdClient is a mock implementation of the statsdClient interface.
type mockStatsdClient struct {
	mock.Mock
}

func (c *mockStatsdClient) TimeInMilliseconds(name string, value float64, tags []string, rate float64) error {
	args := c.Called(name, value, tags, rate)
	return args.Error(0)
}

func (c *mockStatsdClient) Count(name string, value int64, tags []string, rate float64) error {
	args := c.Called(name, value, tags, rate)
	return args.Error(0)
}

func (c *mockStatsdClient) Event(e *statsd.Event) error {
	args := c.Called(e)
	return args.Error(0)
}

type mockLogger struct {
	io.Writer
}

func (l *mockLogger) URL() string {
	return "http://www.google.com"
}
