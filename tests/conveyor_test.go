package conveyor_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
	"github.com/stretchr/testify/assert"
)

// This is just a highlevel sanity test.
func TestConveyor(t *testing.T) {
	checkDocker(t)

	pr, pw := io.Pipe()
	c := newConveyor(t, io.MultiWriter(pw, os.Stdout))

	ctx := context.Background()
	err := c.Push(ctx, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	})
	assert.NoError(t, err)

	w := new(bytes.Buffer)
	_, err = io.Copy(w, pr)
	assert.NoError(t, err)

	if !regexp.MustCompile(`Successfully built`).MatchString(w.String()) {
		t.Log(w.String())
		t.Fatal("Expected image to be built")
	}
}

func TestConveyor_WithTimeout(t *testing.T) {
	checkDocker(t)

	w := new(bytes.Buffer)
	c := newConveyor(t, w)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := c.Push(ctx, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	}); err != nil {
		if _, ok := err.(*builder.BuildCanceledError); !ok {
			t.Fatal("Expected build to be canceled")
		}
	}
}

func newConveyor(t *testing.T, w io.Writer) *conveyor.Conveyor {
	b, err := docker.NewBuilderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	b.DryRun = true

	return conveyor.New(conveyor.Options{
		LogFactory: func(builder.BuildOptions) (builder.Logger, error) {
			return builder.NewLogger(w), nil
		},
		Builder: b,
	})
}

func checkDocker(t testing.TB) {
	if testing.Short() {
		t.Skip("Skipping docker tests because they take a long time")
	}
}
