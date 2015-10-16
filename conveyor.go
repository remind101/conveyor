package conveyor

import (
	"runtime"
	"time"

	"github.com/remind101/conveyor/builder"
)

const (
	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute
)

var (
	// DefaultWorkers is the default number of workers to start.
	DefaultWorkers = runtime.NumCPU()
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
	Workers
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

	workers := NewWorkerPool(numWorkers, WorkerOptions{
		Builder:    b,
		BuildQueue: q,
		LogFactory: options.LogFactory,
	})

	c := &Conveyor{
		BuildQueue: q,
		Workers:    workers,
		builder:    b,
	}

	// Start the workers.
	c.Workers.Start()

	return c
}
