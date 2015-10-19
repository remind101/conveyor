package conveyor

import (
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// BuildQueue represents a queue that can push build requests onto a queue, and
// also pop requests from the queue.
type BuildQueue interface {
	// Push pushes the build request onto the queue.
	Push(context.Context, builder.BuildOptions) error

	// Subscribe returns a channel that can be received on to fetch
	// BuildRequests.
	Subscribe() chan BuildRequest
}

// BuildRequest adds a context.Context to build options.
type BuildRequest struct {
	builder.BuildOptions
	Ctx context.Context
}

// buildQueue is an implementation of the BuildQueue interface that is in memory
// using a channel.
type buildQueue struct {
	queue chan BuildRequest
}

func newBuildQueue(buffer int) *buildQueue {
	return &buildQueue{
		queue: make(chan BuildRequest, buffer),
	}
}

func (q *buildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	q.queue <- BuildRequest{
		Ctx:          ctx,
		BuildOptions: options,
	}
	return nil
}

func (q *buildQueue) Subscribe() chan BuildRequest {
	return q.queue
}
