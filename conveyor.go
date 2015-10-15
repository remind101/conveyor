package conveyor

import (
	"fmt"
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
	BuildQueue

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// New returns a new Conveyor instance.
func New(b builder.Builder) *Conveyor {
	c := &Conveyor{
		Builder:    builder.WithCancel(builder.CloseWriter(b)),
		BuildQueue: newBuildQueue(100),
		Timeout:    DefaultTimeout,
	}

	go c.start()

	return c
}

func (c *Conveyor) start() {
	for {
		ctx, options, err := c.Pop()
		if err != nil {
			log.Println(err)
		}

		_, err = c.Build(ctx, options)
		if err != nil {
			log.Println(err)
		}
	}
}

// Build builds the image.
func (c *Conveyor) Build(ctx context.Context, opts builder.BuildOptions) (image string, err error) {
	w, err := c.newLogger(opts)
	if err != nil {
		return "", err
	}

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
