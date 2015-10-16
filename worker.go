package conveyor

import (
	"log"

	"github.com/remind101/conveyor/builder"
)

// Worker pulls jobs off of a BuildQueue and performs the build.
type Worker struct {
	// Builder to use to build.
	builder.Builder

	// LogFactory to use to build a builder.Logger
	LogFactory builder.LogFactory

	// Queue to pull jobs from.
	buildRequests chan BuildRequest
}

// NewWorker returns a new Worker instance and subscribes to receive build
// requests from the BuildQueue.
func NewWorker(q BuildQueue, b builder.Builder) *Worker {
	return &Worker{
		Builder:       b,
		buildRequests: q.Subscribe(),
	}
}

// Start starts the worker consuming for the BuildQueue.
func (w *Worker) Start() {
	for req := range w.buildRequests {
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
	}
}

func (w *Worker) newLogger(opts builder.BuildOptions) (builder.Logger, error) {
	if w.LogFactory == nil {
		return builder.StdoutLogger(opts)
	}

	return w.LogFactory(opts)
}
