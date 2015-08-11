package conveyor_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
)

// This is just a highlevel sanity test.
func TestConveyor(t *testing.T) {
	checkDocker(t)

	c := newConveyor(t)
	w := &conveyor.NullLogger{}

	if _, err := c.Build(context.Background(), w, conveyor.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	}); err != nil {
		t.Fatal(err)
	}
}

func newConveyor(t *testing.T) *conveyor.Conveyor {
	b, err := conveyor.NewDockerBuilderFromEnv()
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
