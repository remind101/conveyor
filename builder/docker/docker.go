package docker

import (
	"fmt"
	"io"
	"os"
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

const (
	// DefaultBuilderImage is the docker image used to build docker images.
	DefaultBuilderImage = "remind101/conveyor-builder"

	// DefaultDataVolume is the default name of a container serving as a
	// data volume for ssh keys and docker credentials. In general, you
	// shouldn't need to change this.
	DefaultDataVolume = "data"
)

// newUUID returns a new string UUID. This is set to a variable for easy mocking
// in tests.
var newUUID = uuid.New

// dockerClient defines the interface from the go-dockerclient package that we
// use.
type dockerClient interface {
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(docker.RemoveContainerOptions) error
	StartContainer(string, *docker.HostConfig) error
	AttachToContainer(docker.AttachToContainerOptions) error
	StopContainer(string, uint) error
	WaitContainer(string) (int, error)
}

// Builder is a builder.Builder implementation that runs the build in a docker
// container.
type Builder struct {
	// dataVolume is the name of the volume that contains ssh keys and
	// configuration data.
	DataVolume string
	// Name of the image to use to build the docker image. Defaults to
	// DefaultBuilderImage.
	Image string
	// Set to true to enable dry runs. This sets the `DRY` environment
	// variable within the builder container to `true`. The behavior of this
	// flag depends on how the builder image handles the `DRY` environment
	// variable.
	DryRun bool

	client dockerClient
}

// NewBuilder returns a new Builder backed by the docker client.
func NewBuilder(c *docker.Client) *Builder {
	return &Builder{client: c}
}

// NewBuilderFromEnv returns a new Builder with a docker client
// configured from the standard Docker environment variables.
func NewBuilderFromEnv() (*Builder, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return NewBuilder(c), nil
}

// Build executes the docker image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {
	image = fmt.Sprintf("%s:%s", opts.Repository, opts.Sha)
	err = b.build(ctx, w, opts)
	return
}

func (b *Builder) build(ctx context.Context, w io.Writer, opts builder.BuildOptions) error {
	env := []string{
		fmt.Sprintf("REPOSITORY=%s", strings.ToLower(opts.Repository)),
		fmt.Sprintf("BRANCH=%s", opts.Branch),
		fmt.Sprintf("SHA=%s", opts.Sha),
		fmt.Sprintf("DRY=%s", b.dryRun()),
		fmt.Sprintf("CACHE=%s", b.cache(opts)),
	}

	name := strings.Join([]string{
		strings.Replace(opts.Repository, "/", "-", -1),
		opts.Sha,
		newUUID(),
	}, "-")

	c, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Tty:          false,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Image:        b.image(),
			Hostname:     hostname,
			Env:          env,
		},
	})
	if err != nil {
		return fmt.Errorf("create container: %v", err)
	}
	defer b.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            c.ID,
		RemoveVolumes: true,
		Force:         true,
	})

	reporter.AddContext(ctx, "container_id", c.ID)

	if err := b.client.StartContainer(c.ID, &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{b.dataVolume()},
	}); err != nil {
		return fmt.Errorf("start container: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- b.client.AttachToContainer(docker.AttachToContainerOptions{
			Container:    c.ID,
			OutputStream: w,
			ErrorStream:  w,
			Logs:         true,
			Stream:       true,
			Stdout:       true,
			Stderr:       true,
			RawTerminal:  false,
		})
	}()

	var canceled bool
	select {
	case <-ctx.Done():
		// Build was canceled or the build timedout. Stop the container
		// prematurely. We'll SIGTERM and give it 10 seconds to stop,
		// after that we'll SIGKILL.
		if err := b.client.StopContainer(c.ID, 10); err != nil {
			return fmt.Errorf("stop: %v", err)
		}

		// Wait for log streaming to finish.
		if err := <-done; err != nil {
			return fmt.Errorf("attach: %v", err)
		}

		canceled = true
	case err := <-done:
		if err != nil {
			return fmt.Errorf("attach: %v", err)
		}
	}

	exit, err := b.client.WaitContainer(c.ID)
	if err != nil {
		return fmt.Errorf("wait container: %v", err)
	}

	// A non-zero exit status means the build failed.
	if exit != 0 {
		err := fmt.Errorf("container returned a non-zero exit code: %d", exit)
		if canceled {
			err = &builder.BuildCanceledError{
				Err:    err,
				Reason: ctx.Err(),
			}
		}
		return err
	}

	return nil
}

func (b *Builder) dryRun() string {
	if b.DryRun {
		return "true"
	}
	return ""
}

func (b *Builder) image() string {
	if b.Image == "" {
		return DefaultBuilderImage
	}
	return b.Image
}

func (b *Builder) dataVolume() string {
	if b.DataVolume == "" {
		return DefaultDataVolume
	}
	return b.DataVolume
}

func (b *Builder) cache(opts builder.BuildOptions) string {
	if opts.NoCache {
		return "off"
	}

	return "on"
}

var hostname string

func init() {
	hostname, _ = os.Hostname()
}
