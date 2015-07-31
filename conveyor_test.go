package conveyor_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
)

func TestConveyor_Build(t *testing.T) {
	os.Setenv("PATH", fmt.Sprintf("%s:%s", "./bin", os.Getenv("PATH")))
	c := newTestConveyor(t)

	b := new(bytes.Buffer)

	opts := conveyor.BuildOptions{
		Repository:   "ejholmes/captain-test",
		Sha:          "2e4edf57db00d55051c64d1568e2214858a0897d",
		Branch:       "master",
		OutputStream: b,
	}

	if _, err := c.Build(context.Background(), opts); err != nil {
		t.Fatal(err)
	}

	expected := "Built\n"

	if b.String() != expected {
		t.Fatalf("Output => %s; want %s", b.String(), expected)
	}
}

func newTestConveyor(t testing.TB) *conveyor.Conveyor {
	c, err := conveyor.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return c
}
