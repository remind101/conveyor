package conveyor_test

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/docker"
)

// This is just a highlevel sanity test.
func TestConveyor(t *testing.T) {
	checkDocker(t)

	c := newConveyor(t)
	w := new(bytes.Buffer)

	ctx := context.Background()
	if _, err := c.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	}); err != nil {
		t.Log(w.String())
		t.Fatal(err)
	}

	if !regexp.MustCompile(`Successfully built`).MatchString(w.String()) {
		t.Log(w.String())
		t.Fatal("Expected image to be built")
	}
}

func TestConveyor_WithTimeout(t *testing.T) {
	checkDocker(t)

	c := newConveyor(t)
	w := new(bytes.Buffer)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := c.Build(ctx, w, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	}); err != nil {
		if _, ok := err.(*builder.BuildCanceledError); !ok {
			t.Fatal("Expected build to be canceled")
		}
	}
}

func newConveyor(t *testing.T) *conveyor.Conveyor {
	b, err := docker.NewBuilderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	b.DryRun = true
	c := conveyor.New(b)
	return c
}

func checkDocker(t testing.TB) {
	if testing.Short() {
		t.Skip("Skipping docker tests because they take a long time")
	}
}
