package conveyor_test

import (
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
)

func TestConveyor_Build(t *testing.T) {
	c := newTestConveyor(t)
	opts := conveyor.BuildOptions{
		Repository:   "ejholmes/captain-test",
		Sha:          "2e4edf57db00d55051c64d1568e2214858a0897d",
		Branch:       "master",
		OutputStream: os.Stdout,
	}

	if _, err := c.Build(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
}

func newTestConveyor(t testing.TB) *conveyor.Conveyor {
	c, err := conveyor.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return c
}
