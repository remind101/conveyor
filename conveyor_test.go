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
		Commit:     "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
		Branch:     "master",
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
