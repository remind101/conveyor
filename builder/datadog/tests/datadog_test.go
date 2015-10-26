package datadog_test

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/datadog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestBuilder(t *testing.T) {
	c, err := statsd.New("localhost:8125")
	assert.NoError(t, err)

	b := datadog.WithStats(
		builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			return "remind101/acme-inc:1234", nil
		}),
		c,
	)

	_, err = b.Build(context.Background(), ioutil.Discard, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "1234",
	})
	assert.NoError(t, err)
}
