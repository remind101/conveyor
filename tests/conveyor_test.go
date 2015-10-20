package conveyor_test

import (
	"bytes"
	"io"
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/stretchr/testify/assert"
)

// This is just a highlevel sanity test.
func TestConveyor(t *testing.T) {
	checkDocker(t)

	q := conveyor.NewBuildQueue(1)
	l := newLogger()
	w := newWorker(t, q, l)
	w.Start()

	ctx := context.Background()
	err := q.Push(ctx, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	})
	assert.NoError(t, err)

	// Wait for logs to finish writing
	<-l.closed

	if !regexp.MustCompile(`Successfully built`).MatchString(l.String()) {
		t.Log(l.String())
		t.Fatal("Expected image to be built")
	}
}

func TestConveyor_WithTimeout(t *testing.T) {
	checkDocker(t)

	q := conveyor.NewBuildQueue(1)
	l := newLogger()
	w := newWorker(t, q, l)
	w.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := q.Push(ctx, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	}); err != nil {
		if _, ok := err.(*builder.BuildCanceledError); !ok {
			t.Fatal("Expected build to be canceled")
		}
	}
}

func newWorker(t *testing.T, q conveyor.BuildQueue, w io.Writer) conveyor.Workers {
	b, err := docker.NewBuilderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	b.DryRun = true

	return conveyor.NewWorkerPool(1, conveyor.WorkerOptions{
		LogFactory: func(builder.BuildOptions) (builder.Logger, error) {
			return builder.NewLogger(w), nil
		},
		BuildQueue: q,
		Builder:    conveyor.NewBuilder(b),
	})
}

func checkDocker(t testing.TB) {
	if testing.Short() {
		t.Skip("Skipping docker tests because they take a long time")
	}
}

// logger implements the io.Closer interface on top of a bytes.Buffer. It sends
// on the closed channel when Close is called.
type logger struct {
	bytes.Buffer
	closed chan struct{}
}

func newLogger() *logger {
	return &logger{closed: make(chan struct{})}
}

func (l *logger) Close() error {
	close(l.closed)
	return nil
}
