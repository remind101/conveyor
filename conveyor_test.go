package conveyor_test

import (
	"io/ioutil"
	"testing"

	"github.com/remind101/conveyor"
)

func TestConveyor_Build(t *testing.T) {
	c := newTestConveyor(t)
	opts := conveyor.BuildOptions{
		Repository: "remind101/acme-inc",
		Commit:     "72493cc5266a89774dbfe8875790b66cdba15c2e",
		Branch:     "conveyor",
	}

	if err := c.Build(opts); err != nil {
		t.Fatal(err)
	}
}

func newTestConveyor(t testing.TB) *conveyor.Conveyor {
	c, err := conveyor.New()
	if err != nil {
		t.Fatal(err)
	}
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("BuildDir: %s", d)
	c.BuildDir = d
	return c
}
