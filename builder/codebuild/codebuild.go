package codebuild

import (
	"fmt"
	"log"
	"text/template"
	"bytes"
	"strings"
	"io"
	"os"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/remind101/conveyor/builder"
	"golang.org/x/net/context"
)

const (
	// Default Image for codebuild
	DefaultCodebuildImage = "aws/codebuild/docker:1.12.1"

	// Default AWS resource used by codebuild
	DefaultCodebuildComputeType = "BUILD_GENERAL1_SMALL"

)
// Builder is a builder.Builder implementation that runs the build in a docker
// container.
type Builder struct {
	codebuild *codebuild.CodeBuild

	// Required field, arn of the instance-role required by codebuild
	ServiceRole string

	// The Image used by codebuild to build images. Defaults to 
	// DefaultCodebuildImage
	Image string

	// The computing instances AWS CodeBuild will use. Defaults to
	// DefaultCodebuildComputeType
	ComputeType string
}

// NewBuilder returns a new Builder backed by the codebuild client.
func NewBuilder(config client.ConfigProvider) *Builder {
	return &Builder{
		codebuild: codebuild.New(config),
	}
}

// NewBuilderFromEnv returns a new Builder with a codebuild client
// configured from the standard Docker environment variables.
func NewBuilderFromEnv() (*Builder, error) {	

	serviceRole := os.Getenv("CODEBUILD_SERVICE_ROLE")

	if serviceRole == "" {
		return nil, errors.New("CODEBUILD_SERVICE_ROLE must be set when using codebuild builder")
	}

	image := os.Getenv("CODEBUILD_IMAGE")

	if image == "" {
		image = DefaultCodebuildImage
	}

	computeType := os.Getenv("CODEBUILD_COMPUTE_TYPE")

	if computeType == "" {
		computeType = DefaultCodebuildComputeType
	}

	sess := session.Must(session.NewSession())

	return &Builder{
		codebuild: codebuild.New(sess),
		ServiceRole: serviceRole,
		Image: image,
		ComputeType: computeType,
	}, nil
}


func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Build executes the docker image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {
	image = fmt.Sprintf("%s:%s", opts.Repository, opts.Sha)
	err = b.build(ctx, w, opts)
	return
}

// Build executes the codebuild image.
func (b *Builder) build(ctx context.Context, w io.Writer, opts builder.BuildOptions) error {
	
	projectName := strings.Join([]string{
		"conveyor",
		strings.Replace(opts.Repository, "/", "_", -1),
	}, "-")

	startBuild, err := b.startBuild(opts, projectName)

	if err != nil {

		awsErr, ok := err.(awserr.Error)

    	if ok && awsErr.Code() == "ResourceNotFoundException" {

	        createResp, err := b.createProject(opts, projectName)
			check(err)
			log.Printf("%v", createResp)  

			startBuild, err = b.startBuild(opts, projectName)
			check(err)
	        
	    } else {

	    	return err
	   	}

    }

    log.Printf("%v", startBuild)

	return nil
}

func (b *Builder) createProject(opts builder.BuildOptions, projectName string) (resp *codebuild.CreateProjectOutput, err error) {

	log.Printf("Creating new codebuild project")

	githubSource := fmt.Sprintf("https://github.com/%s", opts.Repository)
	
	params := &codebuild.CreateProjectInput{
	    Artifacts: &codebuild.ProjectArtifacts{
	        Type:          aws.String("NO_ARTIFACTS"),
	    },
	    Environment: &codebuild.ProjectEnvironment{
	        ComputeType: aws.String(b.ComputeType),
	        Image:       aws.String(b.Image),
	        Type:        aws.String("LINUX_CONTAINER"),
	    },
	    Name: aws.String(projectName),
	    Source: &codebuild.ProjectSource{ 
	        Type: aws.String("GITHUB"), 
	        Auth: &codebuild.SourceAuth{
	            Type:     aws.String("OAUTH"),
	        },
	        Location:  aws.String(githubSource),
	    },
	    ServiceRole:   aws.String(b.ServiceRole),
	}

	resp, err = b.codebuild.CreateProject(params)
	check(err)
	return
}

func (b *Builder) startBuild(opts builder.BuildOptions, projectName string) (resp *codebuild.StartBuildOutput, err error) {

	log.Printf("Starting codebuild build")

	buildspec, err := b.generateBuildspec(opts)
	check(err)
	fmt.Printf(buildspec)

	params := &codebuild.StartBuildInput{
		ProjectName:   aws.String(projectName),
		SourceVersion: aws.String(opts.Sha),
		BuildspecOverride: aws.String(buildspec),
	}

	resp, err = b.codebuild.StartBuild(params)
	return

}


func (b *Builder) generateBuildspec(opts builder.BuildOptions) (buildspec string, err error) {
	
	specTemplate := `version: 0.1

phases:
  pre_build:
    commands:
      - docker pull "{{.Repository}}:master" || docker pull "{{.Repository}}:latest" || true
  build:
    commands:
      - docker build -t {{.Repository}} .
`

	tmpl, err := template.New("buildspec").Parse(specTemplate)
	check(err)

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, opts)
	check(err)

	buildspec = buf.String()
	return
	
}
