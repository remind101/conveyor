package conveyor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

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
func (c *Conveyor) Build(opts BuildOptions) (err error) {
	defer func() {
		status := "success"
		if err != nil {
			status = "error"
		}
		c.updateStatus(opts.Repository, opts.Commit, status)
	}()

	if err := c.updateStatus(opts.Repository, opts.Commit, "pending"); err != nil {
		return fmt.Errorf("status: %v", err)
	}

	if err := c.checkout(opts); err != nil {
		return fmt.Errorf("checkout: %v", err)
	}

	if err := c.pull(opts); err != nil {
		return fmt.Errorf("pull: %v", err)
	}

	image, err := c.build(opts)
	if err != nil {
		return fmt.Errorf("build: %v", err)
	}

	if err := c.push(opts.Repository); err != nil {
		return fmt.Errorf("push: %v", err)
	}

	if err := c.tag(image.ID, opts.Branch, opts.Commit); err != nil {
		return fmt.Errorf("tag: %v", err)
	}

	return nil
}

// checkout clones the repo and checks out the given commit.
func (c *Conveyor) checkout(opts BuildOptions) error {
	cmd := newCommand("git", "clone", "--depth=50", fmt.Sprintf("--branch=%s", opts.Branch), fmt.Sprintf("git://github.com/%s.git", opts.Repository), opts.Repository)
	cmd.Dir = c.BuildDir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = newCommand("git", "checkout", "-qf", opts.Commit)
	cmd.Dir = filepath.Join(c.BuildDir, opts.Repository)
	return cmd.Run()
}

// pull pulls the last docker image for the branch.
// TODO: try: branch -> latest
func (c *Conveyor) pull(opts BuildOptions) error {
	return c.pullTags(opts.Repository, opts.Branch, "latest")
}

// pullTags attempts to pull each tag. It will return when the first pull
// succeeds or when none of the pulls succeed.
func (c *Conveyor) pullTags(repo string, tags ...string) (err error) {
	for _, t := range tags {
		if err = c.pullTag(repo, t); err != nil {
			if tagNotFound(err) {
				// Try next tag.
				continue
			}
			return
		}
	}

	return
}

func (c *Conveyor) pullTag(repo, tag string) error {
	return c.docker.PullImage(docker.PullImageOptions{
		Repository:   repo,
		Tag:          tag,
		OutputStream: os.Stdout,
	}, c.AuthConfiguration)
}

// build builds the docker image.
func (c *Conveyor) build(opts BuildOptions) (*docker.Image, error) {
	cmd := newCommand("docker", "build", "-t", opts.Repository, ".")
	cmd.Dir = filepath.Join(c.BuildDir, opts.Repository)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return c.docker.InspectImage(opts.Repository)
}

// push pushes the image to the docker registry.
func (c *Conveyor) push(image string) error {
	cmd := newCommand("docker", "push", image)
	return cmd.Run()
}

// tag tags the image id with the given tags.
func (c *Conveyor) tag(id string, tags ...string) error {
	// TODO
	return nil
}

// updateStatus updates the given commit with a new status.
func (c *Conveyor) updateStatus(repo, commit, status string) error {
	return nil
}

// newCommand returns an exec.Cmd that writes to Stdout and Stderr.
func newCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

var tagNotFoundRegex = regexp.MustCompile(`.*Tag (\S+) not found in repository (\S+)`)

func tagNotFound(err error) bool {
	return tagNotFoundRegex.MatchString(err.Error())
}
