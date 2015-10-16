package conveyor

import (
	"fmt"
	"time"

	"github.com/remind101/conveyor/builder"
)

const (
	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute

	// DefaultWorkers is the default number of workers to start.
	DefaultWorkers = 100
)

// Options provided when initializing a new Conveyor instance.
type Options struct {
	// LogFactory used to generate a builder.Logger.
	LogFactory builder.LogFactory

	// The backend used to perform the builds.
	Builder builder.Builder

	// Number of jobs to buffer in the in memory queue.
	Buffer int

	// Number of workers to spin up.
	Workers int
}

// Conveyor is a struct that represents something that can build docker images.
type Conveyor struct {
	BuildQueue
	workers []*Worker
	builder builder.Builder
}

// New returns a new Conveyor instance that spins up multiple workers consuming
// from an in memory queue.
func New(options Options) *Conveyor {
	q := newBuildQueue(options.Buffer)
	b := options.Builder

	numWorkers := options.Workers
	if numWorkers == 0 {
		numWorkers = DefaultWorkers
	}

	var workers []*Worker
	for i := 0; i < numWorkers; i++ {
		w := NewWorker(q, b)
		w.LogFactory = options.LogFactory
		workers = append(workers, w)
	}

	c := &Conveyor{
		BuildQueue: q,
		workers:    workers,
		builder:    b,
	}

	c.Start()

	return c
}

// Start each worker in it's own goroutine.
func (c *Conveyor) Start() {
	for _, w := range c.workers {
		go w.Start()
	}
}

func (c *Conveyor) Cancel() error {
	if b, ok := c.builder.(interface {
		Cancel() error
	}); ok {
		return b.Cancel()
	}

	return fmt.Errorf("Builder does not support Cancel()")
}
