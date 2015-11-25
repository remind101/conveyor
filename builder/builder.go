// Package builder provides methods for building docker images from GitHub
// repositories.
package builder

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

const (
	// Context is used for the commit status context.
	Context = "container/docker"
)

var (
	// ErrShuttingDown can be returned by builders if they're shutting down
	// and not accepting more jobs.
	ErrShuttingDown = errors.New("shutting down")
)

// BuildCanceledError is returned if the build is canceled, or times out and the
// container returns an error.
type BuildCanceledError struct {
	Err    error
	Reason error
}

// Error implements the error interface.
func (e *BuildCanceledError) Error() string {
	return fmt.Sprintf("%s (%s)", e.Err.Error(), e.Reason.Error())
}

// BuildOptions is provided when building an image.
type BuildOptions struct {
	// Repository is the repo to build.
	Repository string
	// Sha is the git commit to build.
	Sha string
	// Branch is the name of the branch that this build relates to.
	Branch string
	// Set to true to disable the layer cache. The zero value is to enable
	// caching.
	NoCache bool
}

// Builder represents something that can build a Docker image.
type Builder interface {
	// Builder should build an image and write output to w.Writer. In general,
	// it's expected that the image will be pushed to some location where it
	// can be pulled by clients.
	//
	// Implementers should take note and handle the ctx.Done() case in the
	// event that the build should timeout or get canceled by the user.
	//
	// The value of image should be the location to pull the immutable
	// image. For example, if the image is built and generates a sha256
	// digest, the value for image may look like:
	//
	//	remind101/acme-inc@sha256:6b558cade79544da908c349ba0e5b63d
	//
	// Or possibly a tag:
	//
	//	remind101/acme-inc:<git sha>
	Build(ctx context.Context, w io.Writer, opts BuildOptions) (image string, err error)
}

// BuilderFunc is a function that implements the Builder interface.
type BuilderFunc func(context.Context, io.Writer, BuildOptions) (string, error)

// Build implements Builder Build.
func (fn BuilderFunc) Build(ctx context.Context, w io.Writer, opts BuildOptions) (string, error) {
	return fn(ctx, w, opts)
}

// statusUpdaterBuilder is a Builder implementation that updates the commit
// status in github.
type statusUpdaterBuilder struct {
	Builder
	github GitHubClient
	since  func(time.Time) time.Duration
}

// UpdateGitHubCommitStatus wraps b to update the GitHub commit status when a
// build starts, and stops.
func UpdateGitHubCommitStatus(b Builder, g GitHubClient) *statusUpdaterBuilder {
	return &statusUpdaterBuilder{
		Builder: b,
		github:  g,
		since:   time.Since,
	}
}

func (b *statusUpdaterBuilder) Build(ctx context.Context, w io.Writer, opts BuildOptions) (image string, err error) {
	t := time.Now()

	defer func() {
		duration := b.since(t)
		description := fmt.Sprintf("Image built in %v.", duration)
		status := "success"
		if err != nil {
			status = "failure"
			description = err.Error()
		}
		b.updateStatus(w, opts, status, description)
	}()

	if err = b.updateStatus(w, opts, "pending", "Image building."); err != nil {
		return
	}

	image, err = b.Builder.Build(ctx, w, opts)
	return
}

// updateStatus updates the given commit with a new status.
func (b *statusUpdaterBuilder) updateStatus(w io.Writer, opts BuildOptions, status string, description string) error {
	context := Context
	parts := strings.SplitN(opts.Repository, "/", 2)

	var desc *string
	if description != "" {
		desc = &description
	}
	var url *string
	if w, ok := w.(interface {
		URL() string
	}); ok {
		url = github.String(w.URL())
	}

	_, _, err := b.github.CreateStatus(parts[0], parts[1], opts.Sha, &github.RepoStatus{
		State:       &status,
		Context:     &context,
		Description: desc,
		TargetURL:   url,
	})
	return err
}

// WithCancel wraps a Builder with a method to stop all builds.
func WithCancel(b Builder) *CancelBuilder {
	return &CancelBuilder{
		Builder: b,
		builds:  make(map[context.Context]context.CancelFunc),
	}
}

type CancelBuilder struct {
	Builder

	sync.Mutex
	stopped bool
	builds  map[context.Context]context.CancelFunc
}

func (b *CancelBuilder) Build(ctx context.Context, w io.Writer, opts BuildOptions) (string, error) {
	if b.stopped {
		return "", ErrShuttingDown
	}

	ctx = b.addBuild(ctx)
	defer b.removeBuild(ctx)

	return b.Builder.Build(ctx, w, opts)
}

func (b *CancelBuilder) Cancel() error {
	b.Lock()

	// Mark as stopped so we don't accept anymore builds.
	b.stopped = true

	// Cancel each build.
	for _, cancel := range b.builds {
		cancel()
	}

	b.Unlock()

	// Wait for all builds to stop.
	for {
		<-time.After(time.Second)

		if len(b.builds) == 0 {
			// All builds stopped.
			break
		}
	}

	return nil
}

func (b *CancelBuilder) addBuild(ctx context.Context) context.Context {
	b.Lock()
	defer b.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	b.builds[ctx] = cancel
	return ctx
}

func (b *CancelBuilder) removeBuild(ctx context.Context) {
	b.Lock()
	defer b.Unlock()

	delete(b.builds, ctx)
}

// CloseWriter wraps a Builder to call Close on w if it implements the io.Closer
// interface.
func CloseWriter(b Builder) Builder {
	return BuilderFunc(func(ctx context.Context, w io.Writer, opts BuildOptions) (image string, err error) {
		defer func() {
			var closeErr error
			if w, ok := w.(io.Closer); ok {
				closeErr = w.Close()
			}
			if err == nil {
				// If there was no error from the builder, let the
				// downstream know that there was an error closing the
				// output stream.
				err = closeErr
			}
		}()

		image, err = b.Build(ctx, w, opts)
		return
	})
}
