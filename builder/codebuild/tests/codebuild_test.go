package codebuild_test

import (
	"flag"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/builder/codebuild"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

var (
	serviceRole = flag.String("test.codebuild.role", "", "Service role to use when testing codebuild")
	dry         = flag.Bool("test.codebuild.dry", true, "When true, enables dry run mode")
)

// This is just a highlevel sanity test.
func TestBuilder_Build(t *testing.T) {
	b := newCodeBuildBuilder(t)
	b.ServiceRole = *serviceRole

	_, err := b.Build(ctx, os.Stdout, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	})
	assert.NoError(t, err)
}

func TestBuilder_Build_Timeout(t *testing.T) {
	b := newCodeBuildBuilder(t)
	b.ServiceRole = *serviceRole

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := b.Build(ctx, os.Stdout, builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
	})
	assert.NoError(t, err)
}

func newCodeBuildBuilder(t *testing.T) *codebuild.Builder {
	if *serviceRole == "" {
		t.Skip("Skipping codebuild integration test because no service role was provided")
	}

	b := codebuild.NewBuilder(session.New())
	b.ServiceRole = *serviceRole
	b.Dry = *dry
	return b
}
