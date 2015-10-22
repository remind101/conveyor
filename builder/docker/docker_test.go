package docker

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	newUUID = func() string {
		return "1234"
	}
}

func TestBuilder_Build(t *testing.T) {
	c := new(mockDockerClient)
	b := &Builder{
		client: c,
	}

	ctx := context.Background()
	w := new(bytes.Buffer)

	c.On("CreateContainer", "remind101-acme-inc-abcd-1234", &docker.Config{
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Image:        DefaultBuilderImage,
		Hostname:     hostname,
		Env: []string{
			"REPOSITORY=remind101/acme-inc",
			"BRANCH=master",
			"SHA=abcd",
			"DRY=",
			"CACHE=on",
		},
	}).Return(&docker.Container{ID: "4321"}, nil)

	c.On("StartContainer", "4321", &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{"data"},
	}).Return(nil)

	c.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4321",
		OutputStream: w,
		ErrorStream:  w,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  false,
	}).Return(nil)

	c.On("WaitContainer", "4321").Return(0, nil)

	c.On("RemoveContainer", docker.RemoveContainerOptions{
		ID:            "4321",
		RemoveVolumes: true,
		Force:         true,
	}).Return(nil)

	_, err := b.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "abcd",
		Branch:     "master",
	})
	assert.NoError(t, err)
}

func TestBuilder_Build_AttachErr(t *testing.T) {
	c := new(mockDockerClient)
	b := &Builder{
		client: c,
	}

	ctx := context.Background()
	w := new(bytes.Buffer)

	c.On("CreateContainer", "remind101-acme-inc-abcd-1234", &docker.Config{
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Image:        DefaultBuilderImage,
		Hostname:     hostname,
		Env: []string{
			"REPOSITORY=remind101/acme-inc",
			"BRANCH=master",
			"SHA=abcd",
			"DRY=",
			"CACHE=on",
		},
	}).Return(&docker.Container{ID: "4321"}, nil)

	c.On("StartContainer", "4321", &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{"data"},
	}).Return(nil)

	c.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4321",
		OutputStream: w,
		ErrorStream:  w,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  false,
	}).Return(errors.New("error attaching to container"))

	c.On("RemoveContainer", docker.RemoveContainerOptions{
		ID:            "4321",
		RemoveVolumes: true,
		Force:         true,
	}).Return(nil)

	_, err := b.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "abcd",
		Branch:     "master",
	})
	assert.Error(t, err, "error attaching to container")
}

func TestBuilder_Build_NonZeroExit(t *testing.T) {
	c := new(mockDockerClient)
	b := &Builder{
		client: c,
	}

	ctx := context.Background()
	w := new(bytes.Buffer)

	c.On("CreateContainer", "remind101-acme-inc-abcd-1234", &docker.Config{
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Image:        DefaultBuilderImage,
		Hostname:     hostname,
		Env: []string{
			"REPOSITORY=remind101/acme-inc",
			"BRANCH=master",
			"SHA=abcd",
			"DRY=",
			"CACHE=on",
		},
	}).Return(&docker.Container{ID: "4321"}, nil)

	c.On("StartContainer", "4321", &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{"data"},
	}).Return(nil)

	c.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4321",
		OutputStream: w,
		ErrorStream:  w,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  false,
	}).Return(nil)

	c.On("WaitContainer", "4321").Return(1, nil)

	c.On("RemoveContainer", docker.RemoveContainerOptions{
		ID:            "4321",
		RemoveVolumes: true,
		Force:         true,
	}).Return(nil)

	_, err := b.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "abcd",
		Branch:     "master",
	})
	assert.Error(t, err, "container returned a non-zero exit code: 1")
}

func TestBuilder_Build_Cancel(t *testing.T) {
	c := new(mockDockerClient)
	b := &Builder{
		client: c,
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := new(bytes.Buffer)

	cancel()

	c.On("CreateContainer", "remind101-acme-inc-abcd-1234", &docker.Config{
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Image:        DefaultBuilderImage,
		Hostname:     hostname,
		Env: []string{
			"REPOSITORY=remind101/acme-inc",
			"BRANCH=master",
			"SHA=abcd",
			"DRY=",
			"CACHE=on",
		},
	}).Return(&docker.Container{ID: "4321"}, nil)

	c.On("StartContainer", "4321", &docker.HostConfig{
		Privileged:  true,
		VolumesFrom: []string{"data"},
	}).Return(nil)

	stopped := make(chan time.Time)
	c.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4321",
		OutputStream: w,
		ErrorStream:  w,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  false,
	}).WaitUntil(stopped).Return(nil)

	c.On("StopContainer", "4321", uint(10)).Run(func(args mock.Arguments) {
		close(stopped)
	}).Return(nil)

	c.On("WaitContainer", "4321").Return(0, nil)

	c.On("RemoveContainer", docker.RemoveContainerOptions{
		ID:            "4321",
		RemoveVolumes: true,
		Force:         true,
	}).Return(nil)

	_, err := b.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Sha:        "abcd",
		Branch:     "master",
	})
	assert.NoError(t, err)
}

// mockDockerClient is a mock implementation of the dockerClient interface.
type mockDockerClient struct {
	mock.Mock
}

func (c *mockDockerClient) CreateContainer(options docker.CreateContainerOptions) (*docker.Container, error) {
	args := c.Called(options.Name, options.Config)
	return args.Get(0).(*docker.Container), args.Error(1)
}

func (c *mockDockerClient) RemoveContainer(options docker.RemoveContainerOptions) error {
	args := c.Called(options)
	return args.Error(0)
}

func (c *mockDockerClient) StartContainer(id string, config *docker.HostConfig) error {
	args := c.Called(id, config)
	return args.Error(0)
}

func (c *mockDockerClient) AttachToContainer(options docker.AttachToContainerOptions) error {
	args := c.Called(options)
	return args.Error(0)
}

func (c *mockDockerClient) StopContainer(id string, timeout uint) error {
	args := c.Called(id, timeout)
	return args.Error(0)
}

func (c *mockDockerClient) WaitContainer(id string) (int, error) {
	args := c.Called(id)
	return args.Int(0), args.Error(1)
}
