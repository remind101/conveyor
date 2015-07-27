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
		Repository:   "remind101/acme-inc",
		Commit:       "72493cc5266a89774dbfe8875790b66cdba15c2e",
		Branch:       "conveyor",
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
