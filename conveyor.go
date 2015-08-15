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

const (
	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute
)

// Conveyor serves as a builder.
type Conveyor struct {
	Builder    builder.Builder
	LogFactory builder.LogFactory

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// New returns a new Conveyor instance.
func New(b builder.Builder) *Conveyor {
	return &Conveyor{
		Builder: builder.WithCancel(builder.CloseWriter(b)),
		Timeout: DefaultTimeout,
	}
}

// EnqueueBuild enqueus a build to run at a later time.
func (c *Conveyor) EnqueueBuild(ctx context.Context, opts builder.BuildOptions) error {
	w, err := c.newLogger(opts)
	if err != nil {
		return err
	}

	go c.Build(ctx, w, opts)
	return err
}

func (c *Conveyor) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {
	log.Printf("Starting build: repository=%s branch=%s sha=%s",
		opts.Repository,
		opts.Branch,
		opts.Sha,
	)

	// Embed the reporter in the context.Context.
	ctx = reporter.WithReporter(ctx, c.reporter())

	if c.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel() // Release resources.
	}

	reporter.AddContext(ctx, "options", opts)
	defer reporter.Monitor(ctx)

	defer func() {
		if err != nil {
			reporter.Report(ctx, err)
		}
	}()

	image, err = c.Builder.Build(ctx, w, opts)
	return
}

func (c *Conveyor) Cancel() error {
	if b, ok := c.Builder.(*builder.CancelBuilder); ok {
		return b.Cancel()
	}

	return fmt.Errorf("Builder does not support Cancel()")
}

func (c *Conveyor) newLogger(opts builder.BuildOptions) (builder.Logger, error) {
	if c.LogFactory == nil {
		return builder.StdoutLogger(opts)
	}

	return c.LogFactory(opts)
}

func (c *Conveyor) reporter() reporter.Reporter {
	if c.Reporter == nil {
		return reporter.ReporterFunc(func(ctx context.Context, err error) error {
			fmt.Fprintf(os.Stderr, "reporting err: %v\n", err)
			return nil
		})
	}

	return c.Reporter
}
