package conveyor

import (
	"fmt"
	"log"
	"strings"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

// Builder represents something that can build a Docker image.
type Builder interface {
	// Build should build the docker image, tag it and push it to the docker
	// registry. This should return the sha256 digest of the image.
	Build(context.Context, BuildOptions) (string, error)
}

// DockerBuilder is a Builder implementation that runs the build in a docker
// container.
type DockerBuilder struct {
	// dataVolume is the name of the volume that contains ssh keys and
	// configuration data.
	DataVolume string
	// Name of the image to use to build the docker image. Defaults to
	// DefaultBuilderImage.
	Image string

	client *docker.Client
}

func NewDockerBuilder(c *docker.Client) *DockerBuilder {
	return &DockerBuilder{client: c}
}

// Build executes the docker image.
func (b *DockerBuilder) Build(ctx context.Context, opts BuildOptions) (string, error) {
	c, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Image:        b.image(),
			Env: []string{
				fmt.Sprintf("REPOSITORY=%s", opts.Repository),
				fmt.Sprintf("BRANCH=%s", opts.Branch),
				fmt.Sprintf("SHA=%s", opts.Sha),
			},
		},
	})
	if err != nil {
		return "", err
	}
	defer b.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            c.ID,
		RemoveVolumes: true,
		Force:         true,
	})

	if err := b.client.StartContainer(c.ID, &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{b.dataVolume()},
	}); err != nil {
		return "", err
	}

	if err := b.client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    c.ID,
		OutputStream: opts.OutputStream,
		ErrorStream:  opts.OutputStream,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}); err != nil {
		return "", err
	}

	// TODO: Return sha256
	return "", nil
}

func (b *DockerBuilder) image() string {
	if b.Image == "" {
		return DefaultBuilderImage
	}
	return b.Image
}

func (b *DockerBuilder) dataVolume() string {
	if b.DataVolume == "" {
		return "data"
	}
	return b.DataVolume
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
		b.updateStatus(opts.Repository, opts.Sha, status)
	}()

	if err = b.updateStatus(opts.Repository, opts.Sha, "pending"); err != nil {
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

func (b *asyncBuilder) Build(ctx context.Context, opts BuildOptions) (string, error) {
	go b.build(ctx, opts)
	return "", nil
}

func (b *asyncBuilder) build(ctx context.Context, opts BuildOptions) {
	if _, err := b.Builder.Build(ctx, opts); err != nil {
		log.Printf("build err: %v", err)
	}
}
