package conveyor

import (
	"fmt"
	"io"
	"log"
	"strings"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/google/go-github/github"

	"golang.org/x/net/context"
)

const (
	// Context is used for the commit status context.
	Context = "container/docker"

	// DefaultBuilderImage is the docker image used to build docker images.
	DefaultBuilderImage = "remind101/conveyor-builder"

	// DefaultDataVolume is the default name of a container serving as a
	// data volume for ssh keys and docker credentials. In general, you
	// shouldn't need to change this.
	DefaultDataVolume = "data"
)

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
	// An io.Writer where output will be written to.
	OutputStream io.Writer
}

// Builder represents something that can build a Docker image.
type Builder interface {
	// Build should build the docker image, tag it and push it to the docker
	// registry. This should return the sha256 digest of the image.
	Build(context.Context, BuildOptions) (string, error)
}

// BuilderFunc is a function that implements the Builder interface.
type BuilderFunc func(context.Context, BuildOptions) (string, error)

func (fn BuilderFunc) Build(ctx context.Context, opts BuildOptions) (string, error) {
	return fn(ctx, opts)
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
	// Set to true to enable dry runs. This sets the `DRY` environment
	// variable within the builder container to `true`. The behavior of this
	// flag depends on how the builder image handles the `DRY` environment
	// variable.
	DryRun bool

	client *docker.Client
}

// NewDockerBuilder returns a new DockerBuilder backed by the docker client.
func NewDockerBuilder(c *docker.Client) *DockerBuilder {
	return &DockerBuilder{client: c}
}

// NewDockerBuilderFromEnv returns a new DockerBuilder with a docker client
// configured from the standard Docker environment variables.
func NewDockerBuilderFromEnv() (*DockerBuilder, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return NewDockerBuilder(c), nil
}

// Build executes the docker image.
func (b *DockerBuilder) Build(ctx context.Context, opts BuildOptions) (string, error) {
	env := []string{
		fmt.Sprintf("REPOSITORY=%s", opts.Repository),
		fmt.Sprintf("BRANCH=%s", opts.Branch),
		fmt.Sprintf("SHA=%s", opts.Sha),
		fmt.Sprintf("DRY=%s", b.dryRun()),
		fmt.Sprintf("CACHE=%s", b.cache(opts)),
	}

	c, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Image:        b.image(),
			Env:          env,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create container: %v", err)
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
		return "", fmt.Errorf("start container: %v", err)
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
		return "", fmt.Errorf("attach: %v", err)
	}

	if w, ok := opts.OutputStream.(io.Closer); ok {
		// Attempt to close the stream if the writer supports it. This
		// is needed for S3 logger to ensure that the file is written.
		if err := w.Close(); err != nil {
			return "", err
		}
	}

	// TODO: Return sha256
	return "", nil
}

func (b *DockerBuilder) dryRun() string {
	if b.DryRun {
		return "true"
	}
	return ""
}

func (b *DockerBuilder) image() string {
	if b.Image == "" {
		return DefaultBuilderImage
	}
	return b.Image
}

func (b *DockerBuilder) dataVolume() string {
	if b.DataVolume == "" {
		return DefaultDataVolume
	}
	return b.DataVolume
}

func (b *DockerBuilder) cache(opts BuildOptions) string {
	if opts.NoCache {
		return "off"
	}

	return "on"
}

// UpdateGitHubCommitStatus wraps b to update the GitHub commit status when a
// build starts, and stops.
func UpdateGitHubCommitStatus(b Builder, g GitHubClient) Builder {
	return &statusUpdaterBuilder{
		Builder: b,
		github:  g,
	}
}

// statusUpdaterBuilder is a Builder implementation that updates the commit
// status in github.
type statusUpdaterBuilder struct {
	Builder
	github GitHubClient
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

// BuildAsync wraps a Builder to run the build in a goroutine.
func BuildAsync(b Builder) Builder {
	build := func(ctx context.Context, opts BuildOptions) {
		if _, err := b.Build(ctx, opts); err != nil {
			log.Printf("build err: %v", err)
		}
	}

	return BuilderFunc(func(ctx context.Context, opts BuildOptions) (string, error) {
		go build(ctx, opts)
		return "", nil
	})
}
