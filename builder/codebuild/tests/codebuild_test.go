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

var (
	ctx = context.Background()

	buildOptions = builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "codebuild-conveyor",
		Sha:        "d0021aa6c6a227e59ac1183e772aee26a659826d",
	}
)

var (
	serviceRole = flag.String("test.codebuild.role", "", "Service role to use when testing codebuild")
	dryRun      = flag.Bool("test.codebuild.dry", true, "When true, enables dry run mode")
)

// This is just a highlevel sanity test.
func TestBuilder_Build(t *testing.T) {
	b := newCodeBuildBuilder(t)
	b.ServiceRole = *serviceRole

	_, err := b.Build(ctx, os.Stdout, buildOptions)
	assert.NoError(t, err)
}

func TestBuilder_Build_Timeout(t *testing.T) {
	b := newCodeBuildBuilder(t)
	b.ServiceRole = *serviceRole

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := b.Build(ctx, os.Stdout, buildOptions)
	assert.NoError(t, err)
}

func newCodeBuildBuilder(t *testing.T) *codebuild.Builder {
	if *serviceRole == "" {
		t.Skip("Skipping codebuild integration test because no service role was provided")
	}

	b := codebuild.NewBuilder(session.New())
	b.ServiceRole = *serviceRole
	b.DryRun = *dryRun
	return b
}
