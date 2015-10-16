package conveyor

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/remind101/conveyor/builder"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// Builder is an implementation of the builder.Builder interface that adds
// timeouts and error reporting.
type Builder struct {
	builder builder.Builder

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// NewBuilder returns a new Builder instance backed by b. It also wraps it with
// cancellation and closes the logs when the build finishes.
func NewBuilder(b builder.Builder) *Builder {
	return &Builder{
		builder: builder.CloseWriter(b),
		Timeout: DefaultTimeout,
	}
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

	image, err = b.builder.Build(ctx, w, opts)
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
