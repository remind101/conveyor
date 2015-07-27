package conveyor

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

// Builder represents something that can build a Docker image.
type Builder interface {
	// Build should build the docker image, tag it and push it to the docker
	// registry. This should return the sha256 digest of the image.
	Build(context.Context, BuildOptions) (string, error)
}

// dockerBuilder is a Builder implementation that shells out to the docker CLI.
type dockerBuilder struct {
	// Name of the image to use to build the docker image. Defaults to
	// DefaultBuilderImage.
	builder string
}

// Build executes the docker image.
func (b *dockerBuilder) Build(ctx context.Context, opts BuildOptions) (string, error) {
	cmd := exec.Command("docker", "run")
	cmd.Stdout = opts.OutputStream
	cmd.Stderr = opts.OutputStream

	if err := cmd.Run(); err != nil {
		return "", err
	}

	// TODO: Return sha256
	return "", nil
}

func (b *dockerBuilder) builderImage() string {
	if b.builder == "" {
		return DefaultBuilderImage
	}
	return b.builder
}

// statusUpdaterBuilder is a Builder implementation that updates the commit
// status in github.
type statusUpdaterBuilder struct {
	Builder
	github githubClient
}

func (b *statusUpdaterBuilder) Build(ctx context.Context, opts BuildOptions) (id string, err error) {
	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		b.updateStatus(opts.Repository, opts.Commit, status)
	}()

	if err = b.updateStatus(opts.Repository, opts.Commit, "pending"); err != nil {
		err = fmt.Errorf("status: %v", err)
		return
	}

	id, err = b.Builder.Build(ctx, opts)
	return
}

// updateStatus updates the given commit with a new status.
func (b *statusUpdaterBuilder) updateStatus(repo, commit, status string) error {
	context := Context
	parts := strings.SplitN(repo, "/", 2)
	_, _, err := b.github.CreateStatus(parts[0], parts[1], commit, &github.RepoStatus{
		State:   &status,
		Context: &context,
	})
	return err
}

// asyncBuilder is an implementation of the Builder interface that builds in a
// goroutine.
type asyncBuilder struct {
	Builder
}

func newAsyncBuilder(b Builder) *asyncBuilder {
	return &asyncBuilder{
		Builder: b,
	}
}

func (b *asyncBuilder) Build(ctx context.Context, opts BuildOptions) error {
	go b.build(ctx, opts)
	return nil
}

func (b *asyncBuilder) build(ctx context.Context, opts BuildOptions) {
	if _, err := b.Builder.Build(ctx, opts); err != nil {
		log.Printf("build err: %v", err)
	}
}
