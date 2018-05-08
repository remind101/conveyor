package worker

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/DataDog/dd-trace-go/tracer"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
)

const (
	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute
)

// Conveyor mocks out the conveyor.Conveyor interface that we use.
type Conveyor interface {
	Writer(ctx context.Context, buildID string) (io.Writer, error)
	BuildStarted(ctx context.Context, buildID string) error
	BuildComplete(ctx context.Context, buildID, image string) error
	BuildFailed(ctx context.Context, buildID string, err error) error
}

// Workers is a collection of workers.
type Workers []*Worker

// Start starts all of the workers in the pool in their own goroutine.
func (w Workers) Start() {
	for _, worker := range w {
		go worker.Start()
	}
}

// Shutdown shuts down all of the workers in the pool.
func (w Workers) Shutdown() error {
	var (
		wg     sync.WaitGroup
		errors []error
	)

	for _, worker := range w {
		wg.Add(1)
		go func(worker *Worker) {
			defer wg.Done()
			if err := worker.Shutdown(); err != nil {
				errors = append(errors, err)
			}
		}(worker)
	}

	wg.Wait()

	if len(errors) == 0 {
		return nil
	}

	return errors[0]
}

// NewPool returns a new set of Worker instances.
func NewPool(c Conveyor, num int, options Options) (workers Workers) {
	for i := 0; i < num; i++ {
		w := New(c, options)
		workers = append(workers, w)
	}
	return
}

// Options are options passed when building a new Worker instance.
type Options struct {
	// Builder to use to perform the builds.
	Builder builder.Builder

	// BuildQueue to pull BuildContexts from.
	BuildRequests chan conveyor.BuildContext
}

// Worker pulls jobs off of a BuildQueue and performs the build.
type Worker struct {
	Conveyor

	// Builder to use to build.
	builder.Builder

	// Queue to pull jobs from.
	buildRequests chan conveyor.BuildContext

	// Channel used to request a shutdown.
	shutdown chan struct{}

	// Channel that is sent on when all builds are finished.
	done chan error
}

// New returns a new Worker instance and subscribes to receive build
// requests from the BuildQueue.
func New(c Conveyor, options Options) *Worker {
	return &Worker{
		Conveyor:      c,
		Builder:       builder.WithCancel(options.Builder),
		buildRequests: options.BuildRequests,
		shutdown:      make(chan struct{}),
		done:          make(chan error),
	}
}

// Start starts the worker consuming for the BuildQueue.
func (w *Worker) Start() error {
	for {
		select {
		case <-w.shutdown:
			var err error
			if b, ok := w.Builder.(interface {
				Cancel() error
			}); ok {
				err = b.Cancel()
			}

			w.done <- err
			break
		case req, ok := <-w.buildRequests:
			if !ok {
				break
			}

			if err := w.build(req.Ctx, req.BuildOptions); err != nil {
				log.Println(err)
			}

			continue
		}

		break
	}

	return nil
}

// build performs a build.
func (w *Worker) build(ctx context.Context, options builder.BuildOptions) (err error) {
	span := tracer.NewRootSpan("Build", "conveyor.worker", options.Repository)
	defer func() { span.FinishWithErr(err) }()

	span.SetMeta("id", options.ID)
	span.SetMeta("repository", options.Repository)
	span.SetMeta("branch", options.Branch)
	span.SetMeta("sha", options.Sha)
	span.SetMeta("no_cache", fmt.Sprintf("%t", options.NoCache))

	ctx = span.Context(ctx)

	buildID := options.ID

	err = w.BuildStarted(ctx, buildID)
	if err != nil {
		return
	}

	var image string
	defer func() {
		if err == nil {
			err = w.BuildComplete(ctx, buildID, image)
		} else {
			w.BuildFailed(ctx, buildID, err)
		}
	}()

	var logger io.Writer
	// Get an io.Writer to write build logs to.
	logger, err = w.Writer(ctx, buildID)
	if err != nil {
		return
	}

	// Perform the build.
	image, err = w.Build(ctx, logger, options)
	if err != nil {
		return
	}

	return
}

// Shutdown stops this worker for processing any build requests. If the Builder
// supports the Cancel method, this function will block until all currently
// processesing builds have been canceled.
func (w *Worker) Shutdown() error {
	close(w.shutdown)
	return <-w.done
}
