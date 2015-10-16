package conveyor

import (
	"log"
	"sync"

	"github.com/remind101/conveyor/builder"
)

// Workers is a collection of workers.
type Workers []*Worker

// Start starts all of the workers in the pool.
func (w Workers) Start() {
	var wg sync.WaitGroup

	for _, worker := range w {
		wg.Add(1)
		go func(worker *Worker) {
			defer wg.Done()
			worker.Start()
		}(worker)
	}

	wg.Wait()
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

// Worker pulls jobs off of a BuildQueue and performs the build.
type Worker struct {
	// Builder to use to build.
	builder.Builder

	// LogFactory to use to build a builder.Logger
	LogFactory builder.LogFactory

	// Queue to pull jobs from.
	buildRequests chan BuildRequest

	// Channel used to request a shutdown.
	shutdown chan struct{}

	// Channel that is sent on when all builds are finished.
	done chan error
}

// NewWorker returns a new Worker instance and subscribes to receive build
// requests from the BuildQueue.
func NewWorker(q BuildQueue, b builder.Builder) *Worker {
	return &Worker{
		Builder:       builder.WithCancel(b),
		buildRequests: q.Subscribe(),
		shutdown:      make(chan struct{}),
		done:          make(chan error),
	}
}

// Start starts the worker consuming for the BuildQueue.
func (w *Worker) Start() {
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
}

// Shutdown stops this worker for processing any build requests. If the Builder
// supports the Cancel method, this function will block until all currently
// processesing builds have been canceled.
func (w *Worker) Shutdown() error {
	close(w.shutdown)
	return <-w.done
}

func (w *Worker) newLogger(opts builder.BuildOptions) (builder.Logger, error) {
	if w.LogFactory == nil {
		return builder.StdoutLogger(opts)
	}

	return w.LogFactory(opts)
}
