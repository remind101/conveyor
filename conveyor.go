package conveyor

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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

	Builds *BuildsService

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// New returns a new Conveyor instance.
func New(b builder.Builder) *Conveyor {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&Build{})

	return &Conveyor{
		Builder: builder.WithCancel(builder.CloseWriter(b)),
		Builds:  &BuildsService{db: &db},
		Timeout: DefaultTimeout,
	}
}

func (c *Conveyor) Build(ctx context.Context, opts builder.BuildOptions) (b *Build, err error) {
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

	var w builder.Logger
	w, err = c.newLogger(opts)
	if err != nil {
		return
	}

	b = &Build{BuildOptions: opts}
	if err = c.Builds.Create(b); err != nil {
		return
	}

	go func() {
		image, err := c.Builder.Build(ctx, w, opts)
		if err != nil {
			reporter.Report(ctx, err)
			return
		}

		b.Image = image
		if err := c.Builds.Update(b); err != nil {
			reporter.Report(ctx, err)
		}
	}()

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
