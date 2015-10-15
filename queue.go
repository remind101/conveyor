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

	// Pop returns the next build request from the queue.
	Pop() (context.Context, builder.BuildOptions, error)
}

type buildRequest struct {
	ctx     context.Context
	options builder.BuildOptions
}

// buildQueue is an implementation of the BuildQueue interface that is in memory
// using a channel.
type buildQueue struct {
	queue chan buildRequest
}

func newBuildQueue(buffer int) *buildQueue {
	return &buildQueue{
		queue: make(chan buildRequest, buffer),
	}
}

func (q *buildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	q.queue <- buildRequest{
		ctx:     ctx,
		options: options,
	}
	return nil
}

func (q *buildQueue) Pop() (context.Context, builder.BuildOptions, error) {
	req := <-q.queue
	return req.ctx, req.options, nil
}
