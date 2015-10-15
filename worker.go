package conveyor

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
	"github.com/remind101/pkg/reporter"
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

// Builder is an implementation of the builder.Builder interface that adds
// timeouts and error reporting.
type Builder struct {
	builder.Builder

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// Build builds the image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {
	log.Printf("Starting build: repository=%s branch=%s sha=%s",
		opts.Repository,
		opts.Branch,
		opts.Sha,
	)

	// Embed the reporter in the context.Context.
	ctx = reporter.WithReporter(ctx, b.reporter())

	if b.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.Timeout)
		defer cancel() // Release resources.
	}

	reporter.AddContext(ctx, "options", opts)
	defer reporter.Monitor(ctx)

	defer func() {
		if err != nil {
			reporter.Report(ctx, err)
		}
	}()

	image, err = b.Builder.Build(ctx, w, opts)
	return
}

func (b *Builder) reporter() reporter.Reporter {
	if b.Reporter == nil {
		return reporter.ReporterFunc(func(ctx context.Context, err error) error {
			fmt.Fprintf(os.Stderr, "reporting err: %v\n", err)
			return nil
		})
	}

	return b.Reporter
}
