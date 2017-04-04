package codebuild

import (
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

// Builder is a builder.Builder implementation that runs the build in a docker
// container.
type Builder struct {
	codebuild *codebuild.CodeBuild
}

// NewBuilder returns a new Builder backed by the docker client.
func NewBuilder(config client.ConfigProvider) *Builder {
	return &Builder{
		codebuild: codebuild.New(config),
	}
}

// NewBuilderFromEnv returns a new Builder with a docker client
// configured from the standard Docker environment variables.
func NewBuilderFromEnv() (*Builder, error) {
	return &Builder{}, nil
}

// Build executes the docker image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {

	log.Printf("CODEBUILD BUILD STARTED")

	params := &codebuild.StartBuildInput{
		ProjectName:   aws.String("codebuild_test"),
		SourceVersion: aws.String(opts.Sha),
	}

	resp, err := b.codebuild.StartBuild(params)

	log.Printf("%v", resp)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	image = fmt.Sprintf("%s:%s", opts.Repository, opts.Sha)

	return
}

func (b *Builder) build(ctx context.Context, w io.Writer, opts builder.BuildOptions) error {
	return nil
}

func (b *Builder) dryRun() string {
	return "test"
}

func (b *Builder) image() string {
	return "test"
}

func (b *Builder) dataVolume() string {
	return "test"
}

func (b *Builder) cache(opts builder.BuildOptions) string {
	return "test"
}
