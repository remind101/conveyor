package conveyor

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/google/go-github/github"
	"github.com/remind101/pkg/reporter"

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

	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute
)

// BuildCanceledError is returned if the build is canceled, or times out and the
// container returns an error.
type BuildCanceledError struct {
	Err error
}

// Error implements the error interface.
func (e *BuildCanceledError) Error() string {
	return e.Err.Error()
}

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
	// Build should build the docker image, tag it and push it to the docker
	// registry. This should return the sha256 digest of the image.
	Build(context.Context, Logger, BuildOptions) (string, error)
}

// BuilderFunc is a function that implements the Builder interface.
type BuilderFunc func(context.Context, Logger, BuildOptions) (string, error)

func (fn BuilderFunc) Build(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
	return fn(ctx, w, opts)
}

// Conveyor serves as a builder.
type Conveyor struct {
	Builder

	// A Reporter to use to report errors.
	Reporter reporter.Reporter

	// Timeout controls how long to wait before canceling a build. A timeout
	// of 0 means no timeout.
	Timeout time.Duration
}

// New returns a new Conveyor instance.
func New(b Builder) *Conveyor {
	return &Conveyor{
		Builder: b,
		Timeout: DefaultTimeout,
	}
}

func (c *Conveyor) Build(ctx context.Context, w Logger, opts BuildOptions) (id string, err error) {
	log.Printf("Starting build: repository=%s branch=%s sha=%s",
		opts.Repository,
		opts.Branch,
		opts.Sha,
	)

	// Embed the reporter in the context.Context.
	ctx = reporter.WithReporter(ctx, c.reporter())

	if c.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel() // Release resources.
	}

	reporter.AddContext(ctx, "options", opts)
	defer reporter.Monitor(ctx)

	defer func() {
		if err != nil {
			reporter.Report(ctx, err)
		}
	}()

	id, err = c.build(ctx, w, opts)
	return
}

// Build performs the build and ensures that the output stream is closed.
func (c *Conveyor) build(ctx context.Context, w Logger, opts BuildOptions) (id string, err error) {
	defer func() {
		var closeErr error
		if w != nil {
			closeErr = w.Close()
		}
		if err == nil {
			// If there was no error from the builder, let the
			// downstream know that there was an error closing the
			// output stream.
			err = closeErr
		}
	}()

	id, err = c.Builder.Build(ctx, w, opts)
	return
}

func (c *Conveyor) reporter() reporter.Reporter {
	if c.Reporter == nil {
		return reporter.ReporterFunc(func(ctx context.Context, err error) error {
			fmt.Fprintf(os.Stderr, "reporting err: %v", err)
			return nil
		})
	}

	return c.Reporter
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
func (b *DockerBuilder) Build(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
	env := []string{
		fmt.Sprintf("REPOSITORY=%s", opts.Repository),
		fmt.Sprintf("BRANCH=%s", opts.Branch),
		fmt.Sprintf("SHA=%s", opts.Sha),
		fmt.Sprintf("DRY=%s", b.dryRun()),
		fmt.Sprintf("CACHE=%s", b.cache(opts)),
	}

	name := strings.Join([]string{
		strings.Replace(opts.Repository, "/", "-", -1),
		opts.Sha,
		uuid.New(),
	}, "-")

	c, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Image:        b.image(),
			Hostname:     hostname,
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

	reporter.AddContext(ctx, "container_id", c.ID)

	if err := b.client.StartContainer(c.ID, &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{b.dataVolume()},
	}); err != nil {
		return "", fmt.Errorf("start container: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		err := b.client.AttachToContainer(docker.AttachToContainerOptions{
			Container:    c.ID,
			OutputStream: w,
			ErrorStream:  w,
			Logs:         true,
			Stream:       true,
			Stdout:       true,
			Stderr:       true,
			RawTerminal:  true,
		})
		done <- err
	}()

	var canceled bool
	select {
	case <-ctx.Done():
		// Build was canceled or the build timedout. Stop the container
		// prematurely. We'll SIGTERM and give it 10 seconds to stop,
		// after that we'll SIGKILL.
		if err := b.client.StopContainer(c.ID, 10); err != nil {
			return "", fmt.Errorf("stop: %v", err)
		}

		// Wait for log streaming to finish.
		err := <-done
		if err != nil {
			return "", fmt.Errorf("attach: %v", err)
		}

		canceled = true
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("attach: %v", err)
		}
	}

	exit, err := b.client.WaitContainer(c.ID)
	if err != nil {
		return "", fmt.Errorf("wait container: %v", err)
	}

	// A non-zero exit status means the build failed.
	if exit != 0 {
		err := fmt.Errorf("container returned a non-zero exit code: %d", exit)
		if canceled {
			err = &BuildCanceledError{
				Err: err,
			}
		}
		return "", err
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

func (b *statusUpdaterBuilder) Build(ctx context.Context, w Logger, opts BuildOptions) (id string, err error) {
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

	id, err = b.Builder.Build(ctx, w, opts)
	return
}

// updateStatus updates the given commit with a new status.
func (b *statusUpdaterBuilder) updateStatus(w Logger, opts BuildOptions, status string, description string) error {
	context := Context
	parts := strings.SplitN(opts.Repository, "/", 2)

	var desc *string
	if description != "" {
		desc = &description
	}
	var url *string
	if status == "success" || status == "failure" || status == "error" {
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

// BuildAsync wraps a Builder to run the build in a goroutine.
func BuildAsync(b Builder) Builder {
	build := func(ctx context.Context, w Logger, opts BuildOptions) {
		if _, err := b.Build(ctx, w, opts); err != nil {
			log.Printf("build err: %v", err)
		}
	}

	return BuilderFunc(func(ctx context.Context, w Logger, opts BuildOptions) (string, error) {
		go build(ctx, w, opts)
		return "", nil
	})
}

var hostname string

func init() {
	hostname, _ = os.Hostname()
}
