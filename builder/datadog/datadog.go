// Package datadog provides middleware that will send events and timings for
// image builds to datadog.
package datadog

import (
	"fmt"
	"io"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// since is a function that calculates the time between now and some other time.
// It's a variable so it can be mocked in tests.
var since = time.Since

// statsdClient represents a client that can send timings and events to
// datadog.
type statsdClient interface {
	Count(name string, value int64, tags []string, rate float64) error
	TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
	Event(*statsd.Event) error
}

// Builder is an implementation of the builder.Builder interface.
type Builder struct {
	builder.Builder
	statsd statsdClient
}

// WithStats returns a new Builder instance that will use the given statsd
// client.
func WithStats(b builder.Builder, c *statsd.Client) *Builder {
	return &Builder{
		Builder: b,
		statsd:  c,
	}
}

// Build performs the build using the underlying Builder and tracks how long it
// took.
func (b *Builder) Build(ctx context.Context, w io.Writer, options builder.BuildOptions) (image string, err error) {
	tags := []string{
		fmt.Sprintf("repo:%s", options.Repository),
	}
	start := time.Now()

	defer func() {
		d := since(start)
		if err != nil {
			_ = b.statsd.Count("conveyor.build.error", 1, tags, 1)
		} else {
			_ = b.statsd.TimeInMilliseconds("conveyor.build.time", d.Seconds()*1000, tags, 1)
			_ = b.statsd.Event(&statsd.Event{
				Title: fmt.Sprintf("Conveyor built %s", image),
				Tags: append(tags,
					fmt.Sprintf("branch:%s", options.Branch),
					fmt.Sprintf("sha:%s", options.Sha),
					fmt.Sprintf("image:%s", image),
				),
			})
		}
	}()

	image, err = b.Builder.Build(ctx, w, options)
	return
}
