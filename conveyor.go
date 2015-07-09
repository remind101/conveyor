package conveyor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
)

type BuildOptions struct {
	// Repository is the repo to build.
	Repository string
	// Commit is the git commit to build.
	Commit string
	// Branch is the name of the branch that this build relates to.
	Branch string
}

type Conveyor struct {
	// BuildDir is the directory where repositories will be cloned.
	BuildDir string
	// AuthConfiguration is the docker authentication credentials for
	// pushing and pulling images from the registry.
	AuthConfiguration docker.AuthConfiguration
	// docker client for interacting with the docker daemon api.
	docker *docker.Client
}

// New returns a new Conveyor instance.
func New() (*Conveyor, error) {
	c, err := dockerutil.NewDockerClientFromEnv()
	if err != nil {
		return nil, err
	}

	return &Conveyor{
		docker: c,
	}, nil
}

// Build builds a docker image for the
func (c *Conveyor) Build(opts BuildOptions) error {
	if err := c.checkout(opts); err != nil {
		return fmt.Errorf("checkout: %v", err)
	}

	if err := c.pull(opts); err != nil {
		return fmt.Errorf("pull: %v", err)
	}

	if err := c.build(opts); err != nil {
		return fmt.Errorf("build: %v", err)
	}
	// Build the docker image.
	// Push the image to the registry.
	// Tag the image with the git commit and branch.
	// Update git commit status.
	return nil
}

// checkout clones the repo and checks out the given commit.
func (c *Conveyor) checkout(opts BuildOptions) error {
	cmd := exec.Command("git", "clone", "--depth=50", fmt.Sprintf("--branch=%s", opts.Branch), fmt.Sprintf("git://github.com/%s.git", opts.Repository), opts.Repository)
	cmd.Dir = c.BuildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "checkout", "-qf", opts.Commit)
	cmd.Dir = filepath.Join(c.BuildDir, opts.Repository)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pull pulls the last docker image for the branch.
// TODO: try: branch -> latest
func (c *Conveyor) pull(opts BuildOptions) error {
	return c.docker.PullImage(
		docker.PullImageOptions{
			Repository:   opts.Repository,
			Tag:          opts.Branch,
			OutputStream: os.Stdout,
		},
		c.AuthConfiguration,
	)
}

// build builds the docker image.
// TODO: Use the docker client to perform the build. We don't use it because the
// docker client handles ignored files: https://github.com/docker/docker/blob/cab02a5bbcb6f6bc0d1cfd61820a8134bd5ed525/api/client/build.go
func (c *Conveyor) build(opts BuildOptions) error {
	cmd := exec.Command("docker", "build", "-t", opts.Repository, ".")
	cmd.Dir = filepath.Join(c.BuildDir, opts.Repository)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
