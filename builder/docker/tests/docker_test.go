package docker_test

import (
	"bytes"
	"regexp"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/stretchr/testify/assert"
)

// This is just a highlevel sanity test.
func TestBuilder_Build(t *testing.T) {
	b := newDockerBuilder(t)

	ctx := context.Background()
	w := new(bytes.Buffer)

	_, err := b.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	})
	assert.NoError(t, err)

	if !regexp.MustCompile(`Successfully built`).MatchString(w.String()) {
		t.Log(w.String())
		t.Fatal("Expected image to be built")
	}
}

func TestBuilder_Build_Cancel(t *testing.T) {
	b := newDockerBuilder(t)

	ctx, cancel := context.WithCancel(context.Background())
	w := new(bytes.Buffer)

	errCh := make(chan error)
	go func() {
		_, err := b.Build(ctx, w, builder.BuildOptions{
			Repository: "remind101/acme-inc",
			Branch:     "master",
			Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
		})
		errCh <- err
	}()

	// Cancel the build and wait for it to tear down the container.
	cancel()
	err := <-errCh

	if _, ok := err.(*builder.BuildCanceledError); !ok {
		t.Fatal("Expected build to be canceled")
	}
}

func newDockerBuilder(t *testing.T) *docker.Builder {
	checkDocker(t)

	b, err := docker.NewBuilderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func checkDocker(t testing.TB) {
	if testing.Short() {
		t.Skip("Skipping docker tests because they take a long time")
	}
}
