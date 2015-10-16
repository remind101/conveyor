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
	BuildQueue
}

// Start starts the worker consuming for the BuildQueue.
func (w *Worker) Start() {
	for {
		ctx, options, err := w.Pop()
		if err != nil {
			log.Println(err)
			continue
		}

		logger, err := w.newLogger(options)
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = w.Build(ctx, logger, options)
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
