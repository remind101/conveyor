package codebuild

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/pkg/base62"
	"github.com/remind101/conveyor/pkg/cloudwatch"
	"golang.org/x/net/context"
)

const (
	// Default Image for codebuild
	DefaultCodebuildImage = "remind101/conveyor-builder:codebuild"

	// Default AWS resource used by codebuild
	DefaultCodebuildComputeType = "BUILD_GENERAL1_SMALL"
)

const (
	BatchGetBuildsWaitTime = 5 * time.Second

	// Maximum amount of time to wait for a build to complete.
	BuildCompleteTimeout = 20 * time.Minute
)

var failedStatuses = []string{
	codebuild.StatusTypeFailed,
	codebuild.StatusTypeFault,
	codebuild.StatusTypeTimedOut,
}

type codebuildClient interface {
	StartBuild(*codebuild.StartBuildInput) (*codebuild.StartBuildOutput, error)
	StopBuild(*codebuild.StopBuildInput) (*codebuild.StopBuildOutput, error)
	CreateProject(*codebuild.CreateProjectInput) (*codebuild.CreateProjectOutput, error)
	BatchGetBuilds(*codebuild.BatchGetBuildsInput) (*codebuild.BatchGetBuildsOutput, error)
}

var projectNameRegex = regexp.MustCompile("[^A-Za-z0-9\\-_]")

// ProjectName converts a repository name to a string that can be used as a
// project name, according to the pattern in
// https://docs.aws.amazon.com/codebuild/latest/APIReference/API_CreateProject.html#API_CreateProject_RequestSyntax
func CleanProjectName(projectName string) string {
	return projectNameRegex.ReplaceAllString(projectName, "-")
}

func ProjectName(opts builder.BuildOptions) string {
	h := fnv.New64()
	h.Write([]byte(opts.Repository))
	hash := base62.Encode(h.Sum64())
	return CleanProjectName(fmt.Sprintf("%s-%s", opts.Repository, hash))
}

// BuildOptions extends build.BuildOptions to include the CodeBuild project
// name.
type BuildOptions struct {
	builder.BuildOptions

	ProjectName string
	DryRun      bool
	DockerCfg   string
}

// Builder is a builder.Builder implementation that builds Docker images using
// CodeBuild.
type Builder struct {
	// BuildSpec is a template.Template that will be executed to generate a
	// buildspec.yml file for CodeBuild. It's given a builder.BuildOptions
	// as data, and is expected to return a valid buildspec.yml that will:
	//
	// 1. Build the Docker Image.
	// 2. Push it to a registry.
	BuildSpec *template.Template

	// Role that will be provided to the CodeBuild project.
	ServiceRole string

	// DryRun specifies whether "Dry" run mode is enabled. When true, the
	// CodeBuild build won't push to the Docker registry.
	DryRun bool

	// DockerCfg can be provided to pass credentials to Docker. This
	// should point to an SSM parameter containing a valid .dockercfg.
	DockerCfg string

	// Function called to determine the CodeBuild project name, given a
	// repository.
	ProjectName func(builder.BuildOptions) string

	codebuild codebuildClient

	// This gets called when we need an io.Reader for a CloudWatch log
	// stream.
	logReader func(group, stream string) (io.ReadCloser, error)
}

func NewBuilder(config client.ConfigProvider) *Builder {
	cloudwatchlogs := cloudwatchlogs.New(config)

	// Use the cloudwatch package to get an io.Reader for a cloudwatch log
	// stream.
	logReader := func(group, stream string) (io.ReadCloser, error) {
		return cloudwatch.NewReader(group, stream, cloudwatchlogs), nil
	}

	return &Builder{
		BuildSpec:   template.Must(template.New("buildspec.yml").Parse(DefaultBuildspec)),
		ProjectName: ProjectName,
		DockerCfg:   "conveyor.dockercfg",
		codebuild:   codebuild.New(config),
		logReader:   logReader,
	}
}

// Build executes the docker image.
func (b *Builder) Build(ctx context.Context, w io.Writer, opts builder.BuildOptions) (image string, err error) {
	projectName := b.ProjectName(opts)
	image = fmt.Sprintf("%s:%s", opts.Repository, opts.Sha)
	err = b.build(ctx, w, BuildOptions{
		BuildOptions: opts,
		ProjectName:  projectName,
		DryRun:       b.DryRun,
		DockerCfg:    b.DockerCfg,
	})
	return
}

func (b *Builder) build(ctx context.Context, w io.Writer, opts BuildOptions) error {
	resp, err := b.startBuild(ctx, opts)
	if err != nil {
		// if the project does not exist, create it.
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			if _, err := b.createProject(ctx, opts); err != nil {
				return err
			}

			resp, err = b.startBuild(ctx, opts)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("starting build for %s: %v", opts.Repository, err)
		}
	}

	buildID := resp.Build.Id

	done := make(chan error)
	go func() {
		done <- b.tailBuild(ctx, w, buildID)
	}()

	select {
	case <-ctx.Done():
		// Try cancelling the build. We don't really care about the
		// error here.
		b.codebuild.StopBuild(&codebuild.StopBuildInput{
			Id: buildID,
		})

		// No wait for log streaming and everything to finish up.
		err = <-done
		if err == nil {
			err = ctx.Err()
		}
	case err = <-done:
	}

	if err == context.Canceled || err == context.DeadlineExceeded {
		return &builder.BuildCanceledError{
			Err:    err,
			Reason: ctx.Err(),
		}
	}

	return err
}

func (b *Builder) tailBuild(ctx context.Context, w io.Writer, buildID *string) error {
	fmt.Fprintf(w, "conveyor: Waiting for CodeBuild build %s to start\n", *buildID)

	build, err := b.waitUntilLogsAvailable(ctx, buildID)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "conveyor: CodeBuild build %s started, streaming logs...\n", *buildID)

	r, err := b.logReader(*build.Logs.GroupName, *build.Logs.StreamName)
	if err != nil {
		return fmt.Errorf("opening cloudwatch logs stream for %s: %v", *build.Id, err)
	}

	copyDone := make(chan error)
	go func() {
		_, err := io.Copy(w, r)
		copyDone <- err
	}()

	ctxTimeout, cancel := context.WithTimeout(context.Background(), BuildCompleteTimeout)
	defer cancel()

	build, _ = b.waitUntilBuildComplete(ctxTimeout, build.Id)

	// Close the cloudwatch log stream reader.
	r.Close()

	// Wait for the io.Copy to complete flushing data to w.
	if err := <-copyDone; err != nil {
		return fmt.Errorf("copying logs: %v", err)
	}

	for _, status := range failedStatuses {
		if *build.BuildStatus == status {
			return fmt.Errorf("CodeBuild build %s failed: %s", *buildID, *build.BuildStatus)
		}
	}

	fmt.Fprintf(w, "conveyor: CodeBuild build %s completed\n", *buildID)

	return nil
}

func (b *Builder) waitUntilLogsAvailable(ctx context.Context, buildID *string) (*codebuild.Build, error) {
	input := &codebuild.BatchGetBuildsInput{Ids: []*string{buildID}}
	resp, err := b.waitUntil(ctx, input, func(resp *codebuild.BatchGetBuildsOutput) bool {
		build := resp.Builds[0]
		return build.Logs != nil
	})
	if err != nil {
		return nil, err
	}
	return resp.Builds[0], nil
}

func (b *Builder) waitUntilBuildComplete(ctx context.Context, buildID *string) (*codebuild.Build, error) {
	input := &codebuild.BatchGetBuildsInput{Ids: []*string{buildID}}
	resp, err := b.waitUntil(ctx, input, func(resp *codebuild.BatchGetBuildsOutput) bool {
		build := resp.Builds[0]
		return *build.BuildComplete
	})
	if err != nil {
		return nil, err
	}
	return resp.Builds[0], nil
}

func (b *Builder) waitUntil(ctx context.Context, input *codebuild.BatchGetBuildsInput, fn func(*codebuild.BatchGetBuildsOutput) bool) (*codebuild.BatchGetBuildsOutput, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := b.codebuild.BatchGetBuilds(input)
		if err != nil {
			return resp, err
		}

		if fn(resp) {
			return resp, nil
		}

		time.Sleep(BatchGetBuildsWaitTime)
	}
}

func (b *Builder) startBuild(ctx context.Context, opts BuildOptions) (*codebuild.StartBuildOutput, error) {
	buf := new(bytes.Buffer)
	if err := b.BuildSpec.Execute(buf, opts); err != nil {
		return nil, fmt.Errorf("generating buildspec: %v", err)
	}

	return b.codebuild.StartBuild(&codebuild.StartBuildInput{
		ProjectName:       aws.String(opts.ProjectName),
		SourceVersion:     aws.String(opts.Sha),
		BuildspecOverride: aws.String(buf.String()),
	})
}

func (b *Builder) createProject(ctx context.Context, opts BuildOptions) (*codebuild.CreateProjectOutput, error) {
	image := DefaultCodebuildImage
	computeType := DefaultCodebuildComputeType
	serviceRole := b.ServiceRole
	githubSource := fmt.Sprintf("https://github.com/%s", opts.Repository)

	return b.codebuild.CreateProject(&codebuild.CreateProjectInput{
		Artifacts: &codebuild.ProjectArtifacts{
			Type: aws.String("NO_ARTIFACTS"),
		},
		Environment: &codebuild.ProjectEnvironment{
			ComputeType:    aws.String(computeType),
			Image:          aws.String(image),
			Type:           aws.String("LINUX_CONTAINER"),
			PrivilegedMode: aws.Bool(true),
		},
		Name: aws.String(opts.ProjectName),
		Source: &codebuild.ProjectSource{
			Type: aws.String("GITHUB"),
			Auth: &codebuild.SourceAuth{
				Type: aws.String("OAUTH"),
			},
			Location: aws.String(githubSource),
		},
		ServiceRole: aws.String(serviceRole),
	})
}

const DefaultBuildspec = `version: 0.1
phases:
  install:
    commands:
      - nohup sh /usr/local/bin/dind /usr/local/bin/docker daemon --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 --storage-driver=overlay&
      - timeout 15 sh -c "until docker info; do echo .; sleep 1; done"
  pre_build:
    commands:
{{ if .DockerCfg }}
      - aws ssm get-parameters --name {{.DockerCfg}} --with-decryption --query Parameters[0].Value --output text > $HOME/.dockercfg
{{ end }}
{{ if .NoCache }}
      - echo "Cache disabled. Not pulling."
{{ else }}
      - {{ if .Branch }}docker pull "{{.Repository}}:{{.Branch}}" || {{ end }}docker pull "{{.Repository}}:master" || docker pull "{{.Repository}}:latest" || true
{{ end }}
  build:
    commands:
      - docker build -t "{{.Repository}}" .
      - docker tag "{{.Repository}}" "{{.Repository}}:{{.Sha}}"
{{ if .Branch }}
      - docker tag -f "{{.Repository}}" "{{.Repository}}:{{.Branch}}"
{{ end }}
  post_build:
    commands:
{{ if .DryRun }}
      - echo "Dry run enabled. Not pushing"
{{ else }}
      - docker push "{{.Repository}}:{{.Sha}}"
      - docker push "{{.Repository}}:latest"
{{ if .Branch }}
      - docker push "{{.Repository}}:{{.Branch}}"
{{ end }}
      - echo "Done pushing to docker registry"
{{ end }}
`
