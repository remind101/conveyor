package events

import (
	"io"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
)

// since is a function that calculates the time between now and some other time.
// It's a variable so it can be mocked in tests.
var since = time.Since

// BuildStartedEvent represents the moment a build is started.
type BuildStartedEvent struct {
	BuildOptions builder.BuildOptions
}

// BuildCompletedEvent represents a build that completed (or failed).
type BuildCompletedEvent struct {
	BuildOptions builder.BuildOptions
	Duration     time.Duration
	Image        string
	Err          error
	Logs         string
}

// BuildEvents represents an interface for notifying an external service about
// a build event.
type BuildEvents interface {
	// BuildEvent is called when there is a build event.
	BuildEvent(event interface{}) error
}

// Builder wraps a builder.Builder to send event notifications to external
// services.
type Builder struct {
	builder.Builder

	events BuildEvents
}

func (b *Builder) Build(ctx context.Context, w io.Writer, options builder.BuildOptions) (image string, err error) {
	start := time.Now()

	_ = b.events.BuildEvent(&BuildStartedEvent{
		BuildOptions: options,
	})

	defer func() {
		event := &BuildCompletedEvent{
			BuildOptions: options,
			Duration:     since(start),
			Image:        image,
			Err:          err,
		}

		if l, ok := w.(interface {
			URL() string
		}); ok {
			event.Logs = l.URL()
		}

		_ = b.events.BuildEvent(event)
	}()

	image, err = b.Builder.Build(ctx, w, options)
	return
}
