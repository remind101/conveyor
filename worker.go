package conveyor

import (
	"io"
	"log"
	"sync"

	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/logs"
)

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

// NewWorkerPool returns a new set of Worker instances.
func NewWorkerPool(num int, options WorkerOptions) (workers Workers) {
	for i := 0; i < num; i++ {
		w := NewWorker(options)
		workers = append(workers, w)
	}
	return
}

// WorkerOptions are options passed when building a new Worker instance.
type WorkerOptions struct {
	// Builder to use to perform the builds.
	Builder builder.Builder

	// BuildQueue to pull BuildRequests from.
	BuildRequests chan BuildRequest

	// Logger used to generate an io.Writer for each build.
	Logger logs.Logger
}

// Worker pulls jobs off of a BuildQueue and performs the build.
type Worker struct {
	// Builder to use to build.
	builder.Builder

	// Logger to use to build a builder.Logger
	Logger logs.Logger

	// Queue to pull jobs from.
	buildRequests chan BuildRequest

	// Channel used to request a shutdown.
	shutdown chan struct{}

	// Channel that is sent on when all builds are finished.
	done chan error
}

// NewWorker returns a new Worker instance and subscribes to receive build
// requests from the BuildQueue.
func NewWorker(options WorkerOptions) *Worker {
	return &Worker{
		Builder:       builder.WithCancel(options.Builder),
		Logger:        options.Logger,
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

			logger, err := w.newLogger(req.BuildOptions)
			if err != nil {
				log.Println(err)
				continue
			}

			_, err = w.Build(req.Ctx, logger, req.BuildOptions)
			if err != nil {
				log.Println(err)
				continue
			}

			continue
		}

		break
	}

	return nil
}

// Shutdown stops this worker for processing any build requests. If the Builder
// supports the Cancel method, this function will block until all currently
// processesing builds have been canceled.
func (w *Worker) Shutdown() error {
	close(w.shutdown)
	return <-w.done
}

func (w *Worker) newLogger(opts builder.BuildOptions) (io.Writer, error) {
	l := w.Logger
	if l == nil {
		l = logs.Discard
	}

	return l.Create(opts.ID)
}
